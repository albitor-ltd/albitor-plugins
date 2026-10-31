// Package shell is the API shell: the HTTP server skeleton every
// Go API starts from. It owns the health route, the JSON 404, the request
// logging and the middleware chain, and it is rendered by the api-go plugin,
// so it starts every app the same way.
//
// An app usually adds handlers in internal/app (see app.Routes) rather than here; the plugins it
// composes add middleware and routes through the api.middleware and api.routes slots, which
// router.go places once at compose time. router.go is the only file here that varies per app.
package shell

import (
	"fmt"
	"log/slog"
	"net/http"

	"webapp/internal/apikit"
)

// Middleware wraps the application handler. The first entry of the api.middleware
// slot is the outermost of the chain, after request logging.
type Middleware = func(http.Handler) http.Handler

// Route registers handlers on the mux.
type Route = func(*http.ServeMux)

// middlewareOf and routesOf collect the entries the generated router.go lists
// in its slots. They exist so the slot placeholder can sit inside a call's
// parentheses, which keeps the generated file gofmt-clean whether the slot is
// empty or holds entries.
func middlewareOf(m ...Middleware) []Middleware { return m }

func routesOf(r ...Route) []Route { return r }

// build assembles the handler: request logging outside everything, then the
// middleware chain in slot order, then the mux with the health route, every
// route registration, and a JSON 404 for anything unmatched.
//
// Logging is outside the chain, not part of it, so a request a middleware
// refuses (an unauthenticated one, say) still gets a correlation id and a log
// line. The health route sits on the mux behind the chain; a middleware that
// refuses unauthenticated requests must leave GET /api/health open itself.
func build(logger *slog.Logger, chain []Middleware, routes []Route) (http.Handler, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			apikit.WriteError(w, http.StatusNotFound, "not found")
			return
		}
		apikit.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	for i, register := range routes {
		if register == nil {
			return nil, fmt.Errorf("route registration %d is nil", i)
		}
		register(mux)
	}

	// Anything not matched above is a 404 JSON error.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		apikit.WriteError(w, http.StatusNotFound, "not found")
	})

	var h http.Handler = mux
	for i := len(chain) - 1; i >= 0; i-- {
		if chain[i] == nil {
			return nil, fmt.Errorf("middleware %d is nil", i)
		}
		h = chain[i](h)
	}
	return apikit.RequestLogging(h, logger), nil
}
