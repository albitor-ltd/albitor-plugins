// Package identity is the verified person a request acts for, and nothing
// else. It is the one identity API an app's handlers use, whatever signs
// people in: handlers call Owner and FromContext, never a token or a cookie. Keep the exported names: handlers depend on them.
//
// The package imports nothing from the app or the shell, so the shell (which
// imports the app) and the app's handlers can both depend on it.
package identity

import (
	"context"
	"errors"
	"net/http"
	"os"
	"slices"
)

// AnonymousOwner owns every record when the API runs with no sign-in at all:
// a local run (ALBITOR_LOCAL=1) without the local issuer, and unit tests. A
// deployed function always has sign-in and never acts as this owner: Owner
// refuses there.
const AnonymousOwner = "anonymous"

// ErrNoIdentity is what Owner answers in a deployed function when the request
// carries no verified identity. A handler answers 401 and does nothing else.
var ErrNoIdentity = errors.New("identity: the request has no verified identity")

// deployedMarker is set by the Lambda runtime itself, so configuration cannot
// unset it. Where it is present nothing may act as AnonymousOwner.
const deployedMarker = "AWS_LAMBDA_FUNCTION_NAME"

// Identity is the verified person a request acts for.
type Identity struct {
	Sub   string `json:"sub"`
	Email string `json:"email,omitempty"`
	// EmailVerified is the token's email_verified claim. It is for the sign-in seam's own checks
	// and is not part of the API contract.
	EmailVerified bool     `json:"-"`
	Groups        []string `json:"groups,omitempty"`
}

// InGroup reports whether the identity belongs to the named group.
func (i Identity) InGroup(name string) bool { return slices.Contains(i.Groups, name) }

type contextKey struct{}

// With returns ctx carrying id. The identity middleware calls it after it has
// verified a token; nothing else should.
func With(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// FromContext returns the verified identity on ctx, if there is one.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(contextKey{}).(Identity)
	return id, ok && id.Sub != ""
}

// Owner names the user a request acts for: the verified token's sub. Every
// user-owned store call takes its answer, so records are scoped by user. Never
// take the owner from the request body or a query parameter: that lets one
// user name another.
//
// A deployed function with no verified identity on the request is an error,
// never AnonymousOwner: the sign-in middleware refuses such a request before a
// handler runs, so reaching here without one means a route the middleware did
// not cover, and the handler must answer 401 rather than act for a shared
// owner. Only a local run or a test, which have no sign-in, get AnonymousOwner.
func Owner(r *http.Request) (string, error) {
	if id, ok := FromContext(r.Context()); ok {
		return id.Sub, nil
	}
	if os.Getenv(deployedMarker) != "" {
		return "", ErrNoIdentity
	}
	return AnonymousOwner, nil
}
