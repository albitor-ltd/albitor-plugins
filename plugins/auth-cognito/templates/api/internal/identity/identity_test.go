package identity

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestOwnerIsTheIdentityOnTheRequest(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/items", nil)
	t.Setenv(deployedMarker, "")
	if got, err := Owner(r); err != nil || got != AnonymousOwner {
		t.Fatalf("Owner with no identity outside a deployed function = %q, %v, want %q", got, err, AnonymousOwner)
	}
	r = r.WithContext(With(r.Context(), Identity{Sub: "user-a", Email: "a@example.invalid"}))
	if got, err := Owner(r); err != nil || got != "user-a" {
		t.Fatalf("Owner = %q, %v, want user-a", got, err)
	}
}

// A deployed function never acts as the anonymous owner: with no identity on the request Owner is
// an error, and an identity still wins when there is one.
func TestOwnerFailsInADeployedFunctionWithNoIdentity(t *testing.T) {
	t.Setenv(deployedMarker, "my-app-api")
	r := httptest.NewRequest("GET", "/api/items", nil)
	if got, err := Owner(r); !errors.Is(err, ErrNoIdentity) || got != "" {
		t.Fatalf("Owner with no identity, deployed = %q, %v, want ErrNoIdentity and no owner", got, err)
	}
	// An identity with no sub is no identity.
	empty := r.WithContext(With(r.Context(), Identity{Email: "a@example.invalid"}))
	if got, err := Owner(empty); !errors.Is(err, ErrNoIdentity) || got != "" {
		t.Fatalf("Owner with a sub-less identity, deployed = %q, %v, want ErrNoIdentity", got, err)
	}
	signedIn := r.WithContext(With(r.Context(), Identity{Sub: "user-a"}))
	if got, err := Owner(signedIn); err != nil || got != "user-a" {
		t.Fatalf("Owner with an identity, deployed = %q, %v, want user-a", got, err)
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
