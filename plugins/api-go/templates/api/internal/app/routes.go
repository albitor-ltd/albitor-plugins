// Package app is where this API's product handlers live. It is yours: the
// api-go plugin wrote this file once and will not overwrite it.
package app

import "net/http"

// Routes registers the product handlers. The router in internal/shell
// calls it once at start, after the health route and the plugins' routes, so
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
// An albitor.yaml schedule runs this same binary: each fire arrives as
// POST /events with the body {"schedule": "<name>"}. Register that route here
// when the app declares a schedule, switch on the name, and run the job before
// answering. A 5xx fails the invocation and Lambda retries it; any other status
// counts as success. Answer 400 for a name you do not know.
//
// Every list endpoint answers apikit.Page through apikit.ServeList, so it has a
// limit and a cursor from its first record.
func Routes(r *http.ServeMux) {
	_ = r // no product routes yet
}
