// Starting point (api-go): the router, written once from the api.middleware and api.routes
// slots the plugins composed with it. It is yours from here on: a plugin added later does
// not rewire it. Add handlers in internal/app/routes.go.

package shell

import (
	"log/slog"
	"net/http"

	"webapp/internal/app"
)

// middleware is the api.middleware slot, outermost first. Each entry names a
// func(http.Handler) http.Handler in this package's scope. The placeholder sits
// inside a call's parentheses so the output is gofmt-clean empty or populated.
var middleware = middlewareOf({{albitor.slot.api.middleware}})

// pluginRoutes is the api.routes slot. Each entry names a func(*http.ServeMux)
// in this package's scope.
var pluginRoutes = routesOf({{albitor.slot.api.routes}})

// Handler returns the API handler: the shell's health route and 404, the
// plugins' routes, the app's routes (app.Routes), behind the middleware chain.
func Handler(logger *slog.Logger) (http.Handler, error) {
	return build(logger, middleware, append(append([]Route{}, pluginRoutes...), app.Routes))
}
