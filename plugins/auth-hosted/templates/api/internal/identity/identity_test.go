package identity

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestOwnerIsTheIdentityOnTheRequest(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/items", nil)
	if got := Owner(r); got != AnonymousOwner {
		t.Fatalf("Owner with no identity = %q, want %q", got, AnonymousOwner)
	}
	r = r.WithContext(With(r.Context(), Identity{Sub: "user-a", Email: "a@example.invalid"}))
	if got := Owner(r); got != "user-a" {
		t.Fatalf("Owner = %q, want user-a", got)
	}
}

func TestAnIdentityWithNoSubIsNoIdentity(t *testing.T) {
	ctx := With(context.Background(), Identity{Email: "a@example.invalid"})
	if _, ok := FromContext(ctx); ok {
		t.Fatal("an identity with no sub must not count as signed in")
	}
	if _, ok := FromContext(context.Background()); ok {
		t.Fatal("an empty context has no identity")
	}
}

func TestInGroup(t *testing.T) {
	id := Identity{Sub: "u", Groups: []string{"admin"}}
	if !id.InGroup("admin") || id.InGroup("staff") {
		t.Fatalf("InGroup on %v", id.Groups)
	}
}
