package shell

import (
	"log/slog"
	"net/http"
	"os"

	"webapp/internal/apikit"
	"webapp/internal/identity"
)

// THE API IDENTITY SEAM. It fills two slots of the generated router: IdentityVerify in
// api.middleware and IdentityRoutes in api.routes. Every request needs a verified sign-in, on any
// path, except /api/health, the sign-in calls in publicRoutes and, in posture public, the routes
// AUTH_PUBLIC_ROUTES lists (identity_public.go).
//
// The pieces:
//   - identity_verify.go (this file): the middleware and GET /api/me.
//   - identity_token.go: verifies a Cognito token against the pool's JWKS.
//   - identity_cognito.go: where the pool comes from, the session cookies and refresh.
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

// publicRoutes are the only requests the gate lets through with no session, as "METHOD path": the
// health route and the sign-in calls this plugin's example (identity_flows.go,
// identity_people.go) answers. They carry no session because they are how a person gets one.
// Every other path, under /api/ or not, needs a verified identity, including the rest of
// /api/auth/ (change-password, delete, the People admin): add a route here only when it must work
// for someone who is not signed in. Nothing here matches by prefix.
var publicRoutes = map[string]bool{
	"POST /api/auth/sign-in":      true,
	"POST /api/auth/sign-up":      true,
	"POST /api/auth/confirm":      true,
	"POST /api/auth/resend":       true,
	"POST /api/auth/forgot":       true,
	"POST /api/auth/reset":        true,
	"POST /api/auth/new-password": true,
	"POST /api/auth/sign-out":     true,
	"GET /api/auth/config":        true,
}

// processSignIn is decided once, when the process starts, from its environment
// (see signInFromEnv).
var processSignIn = signInFromEnv(os.Getenv)

// IdentityVerify is the api.middleware entry: it verifies the request's token,
// refuses a request with no verified identity unless it is health or one of
// publicRoutes, and puts the verified identity where identity.Owner reads it.
func IdentityVerify(next http.Handler) http.Handler {
	return newIdentityGate(next, processSignIn, slog.Default())
}

// IdentityRoutes is the api.routes entry: GET /api/me. The sign-in calls under /api/auth/ are
// registered here too, by the app, when it has screens that make them.
func IdentityRoutes(mux *http.ServeMux) {
	registerIdentityRoutes(mux, processSignIn, slog.Default())
}

func registerIdentityRoutes(mux *http.ServeMux, setup signIn, logger *slog.Logger) {
	mux.HandleFunc(mePath, func(w http.ResponseWriter, r *http.Request) {
		serveMe(w, r, setup)
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
	if r.URL.Path == healthPath || g.setup.anonymous {
		g.next.ServeHTTP(w, r)
		return
	}

	// A route the API does not have answers 404 to everyone, as it would with
	// no sign-in: there is nothing behind it to protect, and the deploy's smoke
	// check expects an unknown /api/ path to be a JSON 404. Every route the API
	// does have, wherever it sits, is protected unless publicRoutes names it.
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
	if publicRoutes[r.Method+" "+r.URL.Path] {
		g.next.ServeHTTP(w, r)
		return
	}

	// A public route (AUTH_PUBLIC_ROUTES, posture public) takes identity as optional: with no
	// credential the handler runs anonymous; a credential that is offered is checked like any
	// other, so a bad or expired one is a 401 and never a quiet downgrade.
	if g.setup.public.isPublic(r) {
		if !g.setup.public.capPublicBody(w, r) {
			return
		}
		if !hasCredential(r) {
			g.next.ServeHTTP(w, r)
			return
		}
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
