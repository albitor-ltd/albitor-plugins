package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"webapp/internal/identity"
)

func asOwner(sub string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	return r.WithContext(identity.With(r.Context(), identity.Identity{Sub: sub}))
}

func ownedFor(t *testing.T, s Store, sub string) *Handle {
	t.Helper()
	h, err := Owned(s, asOwner(sub))
	if err != nil {
		t.Fatalf("Owned(%q): %v", sub, err)
	}
	return h
}

func TestOwnedCreateStampsTheOwner(t *testing.T) {
	ctx, s := context.Background(), NewMemory()
	rec, err := ownedFor(t, s, "alice").Create(ctx, "note", "n1", json.RawMessage(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := Key(OwnerPrefix, "alice", "note")
	if rec.PK != want || rec.SK != "n1" || rec.Rev != 1 {
		t.Fatalf("created %+v, want pk %q sk n1 rev 1", rec, want)
	}
	if _, err := s.Get(ctx, want, "n1"); err != nil {
		t.Fatalf("record is not at the owner's key: %v", err)
	}
	if _, err := ownedFor(t, s, "alice").Create(ctx, "note", "n1", json.RawMessage(`{}`)); !errors.Is(err, ErrExists) {
		t.Fatalf("second create = %v, want ErrExists", err)
	}
}

func TestOwnedHidesAnotherOwnersRecords(t *testing.T) {
	ctx, s := context.Background(), NewMemory()
	alice, bob := ownedFor(t, s, "alice"), ownedFor(t, s, "bob")
	if _, err := alice.Create(ctx, "note", "n1", json.RawMessage(`{"v":"alice"}`)); err != nil {
		t.Fatal(err)
	}

	if _, err := bob.Get(ctx, "note", "n1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get = %v, want ErrNotFound", err)
	}
	page, err := bob.List(ctx, "note", Query{Limit: 10})
	if err != nil || len(page.Items) != 0 {
		t.Errorf("List = %v, %v; want empty", page.Items, err)
	}
	if _, err := bob.Update(ctx, "note", "n1", json.RawMessage(`{"v":"bob"}`), 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update = %v, want ErrNotFound", err)
	}
	if err := bob.Delete(ctx, "note", "n1", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete (revision-checked) = %v, want ErrNotFound", err)
	}
	if err := bob.Delete(ctx, "note", "n1", 0); err != nil {
		t.Errorf("Delete (unconditional) = %v, want nil and no effect", err)
	}

	// The same id is free for the second person and the two never meet.
	if _, err := bob.Create(ctx, "note", "n1", json.RawMessage(`{"v":"bob"}`)); err != nil {
		t.Errorf("bob's create of the same id: %v", err)
	}
	got, err := alice.Get(ctx, "note", "n1")
	if err != nil || string(got.Data) != `{"v":"alice"}` || got.Rev != 1 {
		t.Errorf("alice's record changed: %+v, %v", got, err)
	}
}

func TestOwnedListIsScopedAndPages(t *testing.T) {
	ctx, s := context.Background(), NewMemory()
	alice, bob := ownedFor(t, s, "alice"), ownedFor(t, s, "bob")
	for _, id := range []string{"a", "b", "c"} {
		if _, err := alice.Create(ctx, "note", id, json.RawMessage(`{}`)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := bob.Create(ctx, "note", "z", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	// A caller-supplied PK is ignored: the partition is always the owner's.
	other, _ := Key(OwnerPrefix, "bob", "note")
	first, err := alice.List(ctx, "note", Query{PK: other, Limit: 2})
	if err != nil || len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("first page = %+v, %v", first, err)
	}
	second, err := alice.List(ctx, "note", Query{Limit: 2, Cursor: first.NextCursor})
	if err != nil || len(second.Items) != 1 || second.Items[0].SK != "c" {
		t.Fatalf("second page = %+v, %v", second, err)
	}
	// A cursor minted for one person means nothing to another.
	if _, err := bob.List(ctx, "note", Query{Limit: 2, Cursor: first.NextCursor}); !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("foreign cursor = %v, want ErrInvalidCursor", err)
	}
}

func TestOwnedUpdateAndDeleteOwnRecords(t *testing.T) {
	ctx, s := context.Background(), NewMemory()
	h := ownedFor(t, s, "alice")
	if _, err := h.Update(ctx, "note", "n1", json.RawMessage(`{}`), 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update of a missing record = %v, want ErrNotFound", err)
	}
	if _, err := h.Create(ctx, "note", "n1", json.RawMessage(`{"v":1}`)); err != nil {
		t.Fatal(err)
	}
	rec, err := h.Update(ctx, "note", "n1", json.RawMessage(`{"v":2}`), 1)
	if err != nil || rec.Rev != 2 {
		t.Fatalf("Update = %+v, %v", rec, err)
	}
	if _, err := h.Update(ctx, "note", "n1", json.RawMessage(`{"v":3}`), 1); !errors.Is(err, ErrConflict) {
		t.Errorf("stale Update = %v, want ErrConflict", err)
	}
	if err := h.Delete(ctx, "note", "n1", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Get(ctx, "note", "n1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete = %v, want ErrNotFound", err)
	}
}

func TestOwnedRefusesKeysThatClimbOut(t *testing.T) {
	ctx, s := context.Background(), NewMemory()
	h := ownedFor(t, s, "alice")
	if _, err := h.Create(ctx, "note#x", "n1", json.RawMessage(`{}`)); !errors.Is(err, ErrInvalid) {
		t.Errorf("kind with the separator = %v, want ErrInvalid", err)
	}
	if _, err := h.Get(ctx, "note", ""); !errors.Is(err, ErrInvalid) {
		t.Errorf("empty id = %v, want ErrInvalid", err)
	}
	// An owner that would climb into another partition makes no handle at all.
	if _, err := Owned(s, asOwner("alice#note")); !errors.Is(err, ErrNoOwner) {
		t.Errorf("owner with the separator = %v, want ErrNoOwner", err)
	}
}

func TestOwnedNeedsAnOwner(t *testing.T) {
	s := NewMemory()
	if _, err := Owned(s, nil); !errors.Is(err, ErrNoOwner) {
		t.Errorf("nil request = %v, want ErrNoOwner", err)
	}
	if _, err := Owned(nil, asOwner("alice")); !errors.Is(err, ErrNoOwner) {
		t.Errorf("nil store = %v, want ErrNoOwner", err)
	}
	if _, err := ownedBy(s, ""); !errors.Is(err, ErrNoOwner) {
		t.Errorf("empty owner = %v, want ErrNoOwner", err)
	}
	// A request the sign-in middleware verified yields that person.
	if h := ownedFor(t, s, "alice"); h.Owner() != "alice" {
		t.Errorf("Owner = %q", h.Owner())
	}
}

// A deployed function with no identity on the request has no owner: Owned fails with ErrNoOwner
// rather than scoping records to a shared anonymous one.
func TestOwnedRefusesADeployedRequestWithNoIdentity(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-app-api")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := Owned(NewMemory(), r); !errors.Is(err, ErrNoOwner) {
		t.Errorf("deployed, no identity = %v, want ErrNoOwner", err)
	}
}
