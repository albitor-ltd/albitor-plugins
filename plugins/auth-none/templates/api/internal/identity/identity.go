// Package identity is the person a request acts for, and nothing else. It is
// the one identity API an app's handlers use, whatever signs people in:
// handlers call Owner and FromContext. Here there is no sign-in, so every
// request acts for the one fixed owner. Keep the exported names: handlers depend on them.
//
// The package imports nothing from the app or the shell, so the shell (which
// imports the app) and the app's handlers can both depend on it.
package identity

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
)

func init() {
	// The one runtime marker that an app is open to anyone: a log line, since there is no
	// middleware or route to carry one. Whether that is allowed is the policy packs' call.
	slog.Warn("this API has no sign-in; every request acts for the fixed owner")
}

// FixedOwner owns every record of an app with no sign-in.
const FixedOwner = "owner"

// Identity is the person a request acts for.
type Identity struct {
	Sub    string   `json:"sub"`
	Email  string   `json:"email,omitempty"`
	Groups []string `json:"groups,omitempty"`
}

// InGroup reports whether the identity belongs to the named group.
func (i Identity) InGroup(name string) bool { return slices.Contains(i.Groups, name) }

type contextKey struct{}

// With returns ctx carrying id. There is no middleware, so nothing calls it;
// it exists so code and tests written against a signed-in identity package
// compile unchanged.
func With(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// FromContext returns the identity on ctx, if there is one.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(contextKey{}).(Identity)
	return id, ok && id.Sub != ""
}

// Owner names the user a request acts for: always FixedOwner, because nobody
// signs in. Handlers call it exactly as they would with a real sign-in, so adding
// a real sign-in later changes who the owner is and nothing in them. Never take
// the owner from the request body or a query parameter.
func Owner(r *http.Request) string {
	if id, ok := FromContext(r.Context()); ok {
		return id.Sub
	}
	return FixedOwner
}
