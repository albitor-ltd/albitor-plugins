package identity

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestEveryRequestActsForTheFixedOwner(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/items", nil)
	if got := Owner(r); got != FixedOwner {
		t.Fatalf("Owner = %q, want %q", got, FixedOwner)
	}
	// A header naming someone else changes nothing: there is no sign-in to read.
	r.Header.Set("Authorization", "Bearer someone-else")
	r.Header.Set("X-User", "someone-else")
	if got := Owner(r); got != FixedOwner {
		t.Fatalf("Owner with a token = %q, want %q", got, FixedOwner)
	}
}

func TestAnIdentityWithNoSubIsNoIdentity(t *testing.T) {
	if _, ok := FromContext(With(context.Background(), Identity{Email: "a@example.invalid"})); ok {
		t.Fatal("an identity with no sub must not count")
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
