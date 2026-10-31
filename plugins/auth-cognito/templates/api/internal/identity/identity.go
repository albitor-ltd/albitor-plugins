// Package identity is the verified person a request acts for, and nothing
// else. It is the one identity API an app's handlers use, whichever auth
// plugin the app composes: handlers call Owner and FromContext, never a token
// or a cookie. Keep the exported names: handlers depend on them.
//
// The package imports nothing from the app or the shell, so the shell (which
// imports the app) and the app's handlers can both depend on it.
package identity

import (
	"context"
	"net/http"
	"slices"
)

// AnonymousOwner owns every record when the API runs with no sign-in at all:
// a local run (ALBITOR_LOCAL=1) without the local issuer, and unit tests. A
// deployed function always has sign-in and never acts as this owner.
const AnonymousOwner = "anonymous"

// Identity is the verified person a request acts for.
type Identity struct {
	Sub    string   `json:"sub"`
	Email  string   `json:"email,omitempty"`
	Groups []string `json:"groups,omitempty"`
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
func Owner(r *http.Request) string {
	if id, ok := FromContext(r.Context()); ok {
		return id.Sub
	}
	return AnonymousOwner
}
