// Starting point. Keep this test: it proves, against this app's real routes and with two signed-in
// people, that one person cannot read, change, delete or list another's records.
package identity_test

import (
	"context"
	"net/http"
	"testing"

	"webapp/internal/app"
	"webapp/internal/identity"
	"webapp/internal/ownedscope"
)

// asPerson is a test identity: the same identity.With the sign-in middleware calls after it has
// verified a token, so handlers see it through identity.Owner as they would deployed.
func asPerson(name, sub string) ownedscope.Identity {
	return ownedscope.Identity{Name: name, Apply: ownedscope.AsContext(func(ctx context.Context) context.Context {
		return identity.With(ctx, identity.Identity{Sub: sub, Email: name + "@example.test"})
	})}
}

// ownedDataScopeRows is the table: one row per entity the contract (api/openapi.yaml,
// x-data-model) declares per-user. When you add a per-user entity, add its row here in the same
// change; the test fails while any per-user operation in the contract has no row.
//
// A row's Factory makes one record as the first person through the app's own routes and returns
// its id and valid bodies. Nothing here guesses a request body:
//
//	{
//		Entity: "Army",
//		Factory: func(t ownedscope.TB, as *ownedscope.Actor) ownedscope.Record {
//			body := map[string]any{"id": "army-1", "name": "Vanguard"}
//			if res := as.Do(http.MethodPost, "/api/armies", body); res.Status != http.StatusCreated {
//				t.Fatalf("cannot make an army: %d %s", res.Status, res.Body)
//			}
//			return ownedscope.Record{
//				ID:      "army-1",
//				Create:  body,
//				Update:  map[string]any{"id": "army-1", "name": "Renamed"},
//				Collide: map[string]any{"id": "army-1", "name": "Taken over"},
//			}
//		},
//		// A route that is not create, list, read, update or delete is driven by a Probe...
//		Probes: []ownedscope.Probe{{Method: "POST", Path: "/api/armies/{id}/archive"}},
//		// ...A read (GET, only that) may be waived, with a specific reason of three or more words.
//		// Waive: map[string]string{"GET /api/armies": "armies are listed by the reporting service"},
//	},
func ownedDataScopeRows() []ownedscope.Row {
	return nil // no per-user entity yet
}

// ownedDataScopeAdminProbes is one probe per operation the contract declares admin-only
// (x-admin-only: true, or x-required-groups / x-groups naming admin). The second person is not an
// admin and must be refused. An admin-only operation cannot be waived. For example:
//
//	{Method: "DELETE", Path: "/api/admin/armies/{id}", Params: map[string]string{"id": "army-1"}},
func ownedDataScopeAdminProbes() []ownedscope.Probe {
	return nil // no admin-only operation yet
}

// TestOwnedDataScope runs the two-owner contract test over every per-user operation in
// api/openapi.yaml. Set OWNED_DATA_SCOPE_REPORT to a path to also get the JSON report. Never skip it.
func TestOwnedDataScope(t *testing.T) {
	mux := http.NewServeMux()
	app.Routes(mux)
	ownedscope.Run(t, ownedscope.Config{
		Handler: mux,
		First:   asPerson("alice", "00000000-0000-4000-8000-00000000a11c"),
		Second:  asPerson("bob", "00000000-0000-4000-8000-000000000b0b"),
		Rows:    ownedDataScopeRows(),
		// The sign-in routes under /api/auth/ (the People admin, in an invite-only app) are served
		// by internal/shell and tested there (owned_data_scope_test.go in that package, a skill
		// example). The platform still requires them: this only says another package covers them.
		Elsewhere:   []string{"/api/auth/"},
		AdminProbes: ownedDataScopeAdminProbes(),
	})
}
