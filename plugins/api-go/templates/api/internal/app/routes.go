// Package app is where this API's product handlers live. It is the
// app's own.
package app

import "net/http"

// Routes registers the product handlers. The router in internal/shell
// calls it once at start, after the health route and any route groups in router.go, so
// the mux is where a handler goes:
//
//	func Routes(r *http.ServeMux) {
//		r.HandleFunc("GET /api/widgets", listWidgets)
//	}
//
// Write each handler in its own file in this package, import
// webapp/internal/apikit for apikit.WriteJSON, apikit.WriteError and
// apikit.ServeList, and add one line here. There is no init() registration and
// no generator; a handler that is not listed here is not served.
//
// An albitor.yaml schedule or queue runs this same binary: each fire arrives as
// POST /events, with the body {"schedule": "<name>"} for a schedule and SQS's
// {"Records": [...]} for a queue. Register that route here when the app declares
// either, dispatch on the body, and run the job before answering. A 5xx fails the
// invocation and Lambda retries it; any other status counts as success. Answer 400
// for a name you do not know. Jobs must be idempotent. Never start a ticker of your
// own: the platform owns the schedule tick, locally too.
//
// Every list endpoint answers apikit.Page through apikit.ServeList, so it has a
// limit and a cursor from its first record.
func Routes(r *http.ServeMux) {
	_ = r // no product routes yet
}
