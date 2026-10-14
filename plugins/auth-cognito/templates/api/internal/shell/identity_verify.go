package shell

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"webapp/internal/apikit"
	"webapp/internal/identity"
)

// THE API IDENTITY SEAM of the auth-cognito plugin (auth@1). It fills two
// slots of the generated router: IdentityVerify in api.middleware and
// IdentityRoutes in api.routes. Every /api/ request now needs a verified
// sign-in, except /api/health and the sign-in calls themselves.
//
// The pieces:
//   - identity_verify.go (this file): the middleware, GET /api/me, the routes.
//   - identity_token.go: verifies a Cognito token against the pool's JWKS.
//   - identity_cognito.go: where the pool comes from, and POST /api/auth/*:
//     the sign-in, sign-up, confirm-code and reset calls the web app makes.
//
// Handlers never see any of this. They call identity.Owner(r) from
// webapp/internal/identity, which answers the verified token's `sub`.
//
// The session lives in two HttpOnly cookies the API sets, so no token is ever
// readable by the page's JavaScript. A script or another client may send
// `Authorization: Bearer <token>` instead.

// The paths the seam itself answers or leaves open.
const (
	healthPath   = "/api/health"
	mePath       = "/api/me"
	authPathRoot = "/api/auth/"
)

// processSignIn is decided once, when the process starts, from its environment
// (see signInFromEnv).
var processSignIn = signInFromEnv(os.Getenv)

// IdentityVerify is the api.middleware entry: it verifies the request's token,
// refuses an /api/ request with no verified identity, and puts the verified
// identity where identity.Owner reads it.
func IdentityVerify(next http.Handler) http.Handler {
	return newIdentityGate(next, processSignIn, slog.Default())
}

// IdentityRoutes is the api.routes entry: GET /api/me and POST /api/auth/*.
func IdentityRoutes(mux *http.ServeMux) {
	registerIdentityRoutes(mux, processSignIn, slog.Default())
}

func registerIdentityRoutes(mux *http.ServeMux, setup signIn, logger *slog.Logger) {
	mux.HandleFunc(mePath, func(w http.ResponseWriter, r *http.Request) {
		serveMe(w, r, setup)
	})
	mux.HandleFunc(authPathRoot, func(w http.ResponseWriter, r *http.Request) {
		if setup.auth == nil {
			apikit.WriteJSON(w, http.StatusNotImplemented, map[string]string{
				"error": "sign-in is not available in a local run", "code": "local",
			})
			return
		}
		setup.auth.serveAuth(w, r, logger)
	})
}

// serveMe is GET /api/me: who the session belongs to. By the time it runs the
// gate has verified the request, so the identity is on the context; it is
// missing only in a local run with no sign-in at all.
func serveMe(w http.ResponseWriter, r *http.Request, setup signIn) {
	if r.Method != http.MethodGet {
		apikit.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	if id, ok := identity.FromContext(r.Context()); ok {
		apikit.WriteJSON(w, http.StatusOK, id)
		return
	}
	if setup.anonymous {
		apikit.WriteJSON(w, http.StatusOK, map[string]any{"sub": identity.AnonymousOwner, "local": true})
		return
	}
	writeUnauthenticated(w)
}

// identityGate is IdentityVerify's handler.
type identityGate struct {
	next   http.Handler
	setup  signIn
	logger *slog.Logger
}

func newIdentityGate(next http.Handler, setup signIn, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &identityGate{next: next, setup: setup, logger: logger}
}

func (g *identityGate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/") || path == healthPath || g.setup.anonymous {
		g.next.ServeHTTP(w, r)
		return
	}

	// A route the API does not have answers 404 to everyone, as it would with
	// no sign-in: there is nothing behind it to protect, and the deploy's smoke
	// check expects an unknown /api/ path to be a JSON 404.
	if unknownRoute(g.next, r) {
		g.next.ServeHTTP(w, r)
		return
	}

	if g.setup.err != nil {
		// Fail closed: with no pool to check against, nothing is let through.
		g.logger.ErrorContext(r.Context(), "sign-in is not configured",
			slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())),
			slog.String("error", g.setup.err.Error()))
		apikit.WriteError(w, http.StatusServiceUnavailable, "sign-in is unavailable")
		return
	}

	// The sign-in calls are how a person gets a session: they carry none.
	if strings.HasPrefix(path, authPathRoot) {
		g.next.ServeHTTP(w, r)
		return
	}

	id, ok := g.setup.auth.authenticate(w, r, g.logger)
	if !ok {
		return // authenticate has answered: 401, or 503 when the keys are unreachable
	}
	g.next.ServeHTTP(w, r.WithContext(identity.With(r.Context(), id)))
}

// routeMatcher is the part of *http.ServeMux unknownRoute needs.
type routeMatcher interface {
	Handler(r *http.Request) (http.Handler, string)
}

// unknownRoute reports whether next would answer r from its catch-all "/"
// pattern, the shell's JSON 404. Anything else, including a handler this
// cannot inspect (another middleware wrapping the mux), is treated as a real
// route and protected.
func unknownRoute(next http.Handler, r *http.Request) bool {
	m, ok := next.(routeMatcher)
	if !ok {
		return false
	}
	_, pattern := m.Handler(r)
	return pattern == "/"
}
