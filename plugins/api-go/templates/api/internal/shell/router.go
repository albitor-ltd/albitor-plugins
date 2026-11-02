// The router: the middleware chain and the routes added beside the app's own, written once
// when the app was composed. Add handlers in internal/app/routes.go; wire any later
// middleware or route group by hand.

package shell

import (
	"log/slog"
	"net/http"

	"webapp/internal/app"
)

// middleware is the middleware chain, outermost first. Each entry names a
// func(http.Handler) http.Handler in this package's scope. The placeholder sits
// inside a call's parentheses so the output is gofmt-clean empty or populated.
var middleware = middlewareOf({{albitor.slot.api.middleware}})

// groupRoutes are route groups registered beside the app's own. Each entry names a
// func(*http.ServeMux) in this package's scope.
var groupRoutes = routesOf({{albitor.slot.api.routes}})

// Handler returns the API handler: the shell's health route and 404, the
// route groups above, the app's routes (app.Routes), behind the middleware chain.
func Handler(logger *slog.Logger) (http.Handler, error) {
	return build(logger, middleware, append(append([]Route{}, groupRoutes...), app.Routes))
}
