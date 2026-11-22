// Package app is where this API's product handlers live. It is the
// app's own.
package app

import "net/http"

// registrar is one feature's registration function: it takes the mux and registers that
// feature's patterns. The list of them is in registry.go.
type registrar func(*http.ServeMux)

// Routes registers the product handlers. The router in internal/shell calls it once at start,
// after the health route and any route groups in router.go. It only walks registrars (registry.go),
// so it is not edited when a feature is added.
//
// A feature is its own package, internal/<feature>, with an exported Register that takes the mux
// and registers that feature's patterns; registry.go imports it and lists it:
//
//	// internal/widgets/routes.go
//	func Register(r *http.ServeMux) {
//		r.HandleFunc("GET /api/widgets", listWidgets)
//	}
//
// A feature imports shared packages (apikit, identity, a store), never this package and never a
// sibling feature. Import webapp/internal/apikit for apikit.WriteJSON, apikit.WriteError and
// apikit.ServeList in the handler file.
//
// An albitor.yaml schedule or queue runs this same binary: each fire arrives as
// POST /events, with the body {"schedule": "<name>"} for a schedule and SQS's
// {"Records": [...]} for a queue. Register that route from its own feature package when the app
// declares either, dispatch on the body, and run the job before answering. A 5xx fails the
// invocation and Lambda retries it; any other status counts as success. Answer 400
// for a name you do not know. Jobs must be idempotent. Never start a ticker of your
// own: the platform owns the schedule tick, locally too.
//
// Every list endpoint answers apikit.Page through apikit.ServeList, so it has a
// limit and a cursor from its first record.
func Routes(r *http.ServeMux) {
	for _, register := range registrars {
		register(r)
	}
}
