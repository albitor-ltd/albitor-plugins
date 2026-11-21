// Package app is where this API's product handlers live. It is the
// app's own.
package app

import "net/http"

// registrars lists one registration function per feature, one line each, in the order they
// mount. A feature owns a file pair, <feature>.go (handlers) and <feature>_routes.go (its
// register<Feature> function); adding a feature adds exactly one line here and edits no other
// shared file. Keep this list append-only: never reorder, reflow or edit another feature's
// line, so two features landing in parallel resolve a rebase by keeping both lines.
//
//	var registrars = []func(*http.ServeMux){
//		registerWidgets,
//		registerGadgets,
//	}
//
// There is no init() registration and no generator; a feature that is not listed here is not
// served.
var registrars = []func(*http.ServeMux){
	// one line per feature, appended below
}

// Routes registers the product handlers. The router in internal/shell calls it once at start,
// after the health route and any route groups in router.go. It only walks registrars, so it
// is not edited when a feature is added.
//
// A feature's register function takes the mux and registers that feature's patterns:
//
//	// widgets_routes.go
//	func registerWidgets(r *http.ServeMux) {
//		r.HandleFunc("GET /api/widgets", listWidgets)
//	}
//
// Import webapp/internal/apikit for apikit.WriteJSON, apikit.WriteError and
// apikit.ServeList in the handler file.
//
// An albitor.yaml schedule or queue runs this same binary: each fire arrives as
// POST /events, with the body {"schedule": "<name>"} for a schedule and SQS's
// {"Records": [...]} for a queue. Register that route in its own register function when the app
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
