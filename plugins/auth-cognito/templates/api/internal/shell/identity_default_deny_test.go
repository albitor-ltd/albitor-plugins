package shell

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"webapp/internal/apikit"
)

// denyGate is the gate over a mux that serves every route the tests below name, as an app that
// registered them would. It does not use registerIdentityRoutes, so it stands the same with or
// without the plugin's sign-in example.
func denyGate(f *fakeCognito, srv *httptest.Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		apikit.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	for _, pattern := range []string{
		"/api/auth/sign-in", "/api/auth/sign-up", "/api/auth/confirm", "/api/auth/resend", "/api/auth/forgot",
		"/api/auth/reset", "/api/auth/new-password", "/api/auth/sign-out", "/api/auth/config",
		"/api/auth/change-password", "/api/auth/delete", "/api/auth/users", "/api/auth/anything-else",
		"/api/auth/sign-in/", "/admin/report", "/events",
	} {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, _ *http.Request) {
			apikit.WriteJSON(w, http.StatusOK, map[string]string{"reached": "yes"})
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		apikit.WriteError(w, http.StatusNotFound, "not found")
	})
	setup := signIn{auth: &cognitoAuth{
		clientID: testClientID, endpoint: srv.URL + "/", client: srv.Client(),
		verifier: newCognitoVerifier(f.issuer, testClientID, f.issuer+"/.well-known/jwks.json", srv.Client()),
	}}
	return newIdentityGate(mux, setup, identityTestLogger())
}

// The gate is default-deny: a request with no verified identity gets through only to the health
// route and the exact sign-in calls in publicRoutes. Everything else, on any path and under any
// prefix, is refused, so a route that sits outside /api/ (the API Gateway URL has no path filter in
// front of it) or under /api/auth/ is no way round it.
func TestTheGateAllowsOnlyHealthAndTheNamedSignInCalls(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := denyGate(f, srv)

	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/health", http.StatusOK},
		// Every sign-in call the plugin names reaches the handler with no session.
		{http.MethodPost, "/api/auth/sign-in", http.StatusOK},
		{http.MethodPost, "/api/auth/sign-up", http.StatusOK},
		{http.MethodPost, "/api/auth/confirm", http.StatusOK},
		{http.MethodPost, "/api/auth/resend", http.StatusOK},
		{http.MethodPost, "/api/auth/forgot", http.StatusOK},
		{http.MethodPost, "/api/auth/reset", http.StatusOK},
		{http.MethodPost, "/api/auth/new-password", http.StatusOK},
		{http.MethodPost, "/api/auth/sign-out", http.StatusOK},
		{http.MethodGet, "/api/auth/config", http.StatusOK},
		// A registered route outside /api/ is protected, and /events is not open in this lane.
		{http.MethodGet, "/admin/report", http.StatusUnauthorized},
		{http.MethodPost, "/events", http.StatusUnauthorized},
		{http.MethodGet, "/events", http.StatusUnauthorized},
		// An /api/auth/ path the plugin does not name is refused: the prefix opens nothing.
		{http.MethodPost, "/api/auth/change-password", http.StatusUnauthorized},
		{http.MethodPost, "/api/auth/delete", http.StatusUnauthorized},
		{http.MethodGet, "/api/auth/users", http.StatusUnauthorized},
		{http.MethodPost, "/api/auth/anything-else", http.StatusUnauthorized},
		// A named path with another method is not the named call.
		{http.MethodGet, "/api/auth/sign-in", http.StatusUnauthorized},
		{http.MethodPost, "/api/auth/config", http.StatusUnauthorized},
		// Nothing matches by prefix or trailing slash.
		{http.MethodPost, "/api/auth/sign-in/", http.StatusUnauthorized},
		{http.MethodPost, "/api/auth/sign-in/extra", http.StatusUnauthorized},
		// Health is the one path, not a prefix.
		{http.MethodGet, "/api/health/extra", http.StatusNotFound},
		// A path nothing serves is the JSON 404 to everyone, behind nothing worth protecting.
		{http.MethodGet, "/no-such-page", http.StatusNotFound},
		{http.MethodGet, "/api/no-such-route", http.StatusNotFound},
	} {
		rec := authDo(t, h, tc.method, tc.path, "", nil)
		if rec.Code != tc.want {
			t.Errorf("%s %s = %d, want %d: %s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}

// With a verified identity the same routes are served, so the refusals above are about the
// session and not about the route.
func TestADeniedPathIsServedToAVerifiedIdentity(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := denyGate(f, srv)
	for _, path := range []string{"/admin/report", "/events", "/api/auth/change-password", "/api/auth/anything-else"} {
		rec := authDo(t, h, http.MethodGet, path, "", withBearer(f.token("user-a", nil)))
		if rec.Code != http.StatusOK {
			t.Errorf("signed in, GET %s = %d, want 200: %s", path, rec.Code, rec.Body.String())
		}
	}
}

// A deployed function reaching a handler with no identity on the request must not act as the
// anonymous owner: identity.Owner is an error, and the handler answers 401.
func TestADeployedHandlerWithNoIdentityIsRefusedNotAnonymous(t *testing.T) {
	t.Setenv(envLambdaName, "my-app-api")
	// A no-sign-in setup mounts the route table with no gate refusing anyone, as a mis-wired deploy would.
	h := composeIdentity(signIn{anonymous: true})
	rec := authDo(t, h, http.MethodGet, "/api/items", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("a deployed handler with no identity = %d %s, want 401", rec.Code, rec.Body.String())
	}
}
