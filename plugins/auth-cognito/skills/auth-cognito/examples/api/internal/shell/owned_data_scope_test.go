package shell

import (
	"net/http"
	"testing"

	"webapp/internal/ownedscope"
)

// TestOwnedDataScopePeopleAdminRoutes is the two-owner contract test for the routes this package
// serves: the People admin, which the contract declares admin-only. A signed-in person who is not
// in the user-admin group must be answered 403 on every one of them. Keep it with the routes; the
// seeded test in internal/identity covers the app's own routes and leaves these to this package.
func TestOwnedDataScopePeopleAdminRoutes(t *testing.T) {
	f, srv := newFakeCognito(t)
	person := func(name string, groups ...string) ownedscope.Identity {
		claims := map[string]any{}
		if len(groups) > 0 {
			claims["cognito:groups"] = groups
		}
		token := f.token(name, claims)
		return ownedscope.Identity{Name: name, Apply: func(r *http.Request) *http.Request {
			withBearer(token)(r)
			return r
		}}
	}
	body := map[string]string{"email": "a@example.invalid"}
	ownedscope.Run(t, ownedscope.Config{
		Handler: inviteOnlyGate(f, srv, identityTestLogger()),
		// Neither is a user administrator: the second is the one that must be refused.
		First:  person("first"),
		Second: person("second", "editors"),
		AdminProbes: []ownedscope.Probe{
			{Method: "GET", Path: "/api/auth/users"},
			{Method: "POST", Path: "/api/auth/users", Body: body},
			{Method: "POST", Path: "/api/auth/users/disable", Body: body},
			{Method: "POST", Path: "/api/auth/users/enable", Body: body},
			{Method: "POST", Path: "/api/auth/users/reset-password", Body: body},
		},
	})
}
