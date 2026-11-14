package shell

import (
	"net/http"
	"testing"

	"webapp/internal/apikit"
)

// denyGate is the gate over a mux that serves every route the tests below name, as an app that
// registered them would. It does not use registerIdentityRoutes, so it stands the same with or
// without the plugin's sign-in example.
func denyGate(f *fakeCognito) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		apikit.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	for _, pattern := range []string{
		"/api/auth/login", "/api/auth/callback", "/api/auth/sign-out", "/api/auth/delete",
		"/api/auth/anything-else", "/api/auth/login/", "/admin/report", "/events",
	} {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, _ *http.Request) {
			apikit.WriteJSON(w, http.StatusOK, map[string]string{"reached": "yes"})
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		apikit.WriteError(w, http.StatusNotFound, "not found")
	})
	return newIdentityGate(mux, signIn{auth: f.auth()}, identityTestLogger())
}

// A function that runs a schedule or queue has the adapter's pass-through path set to /events, and
// only there does the platform's POST /events need no session. It is that POST alone.
func TestEventsAreOpenOnlyWhereNativeEventsArrive(t *testing.T) {
	t.Setenv(eventsPassThroughEnv, eventsPath)
	h := denyGate(newFakeCognito(t))
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/events", http.StatusOK},
		{http.MethodGet, "/events", http.StatusUnauthorized},
		{http.MethodPut, "/events", http.StatusUnauthorized},
		{http.MethodPost, "/events/extra", http.StatusNotFound}, // nothing serves it
		{http.MethodPost, "/admin/report", http.StatusUnauthorized},
	} {
		if rec := authDo(t, h, tc.method, tc.path, "", nil); rec.Code != tc.want {
			t.Errorf("%s %s = %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}
	// Any other value of the variable opens nothing.
	t.Setenv(eventsPassThroughEnv, "/other")
	if rec := authDo(t, denyGate(newFakeCognito(t)), http.MethodPost, "/events", "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("POST /events with another pass-through path = %d, want 401", rec.Code)
	}
}

// The gate is default-deny: a request with no verified identity gets through only to the health
// route and the exact sign-in routes in publicRoutes. Everything else, on any path and under any prefix, is refused, so
// a route outside /api/ (the API Gateway URL has no path filter in front of it) or under
// /api/auth/ is no way round it.
func TestTheGateAllowsOnlyHealthTheNamedSignInRoutesAndEvents(t *testing.T) {
	h := denyGate(newFakeCognito(t))

	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/health", http.StatusOK},
		{http.MethodGet, "/api/auth/login", http.StatusOK},
		{http.MethodGet, "/api/auth/callback", http.StatusOK},
		{http.MethodPost, "/api/auth/sign-out", http.StatusOK},
		// The HTTP function has no pass-through path, so /events needs a session there.
		{http.MethodPost, "/events", http.StatusUnauthorized},
		// A registered route outside /api/ is protected.
		{http.MethodGet, "/admin/report", http.StatusUnauthorized},
		// /events is the platform's POST alone.
		{http.MethodGet, "/events", http.StatusUnauthorized},
		{http.MethodPut, "/events", http.StatusUnauthorized},
		// An /api/auth/ path the plugin does not name is refused: the prefix opens nothing.
		{http.MethodPost, "/api/auth/delete", http.StatusUnauthorized},
		{http.MethodGet, "/api/auth/delete", http.StatusUnauthorized},
		{http.MethodPost, "/api/auth/anything-else", http.StatusUnauthorized},
		// A named path with another method is not the named route.
		{http.MethodPost, "/api/auth/login", http.StatusUnauthorized},
		{http.MethodGet, "/api/auth/sign-out", http.StatusUnauthorized},
		// Nothing matches by prefix or trailing slash.
		{http.MethodGet, "/api/auth/login/", http.StatusUnauthorized},
		{http.MethodGet, "/api/auth/login/extra", http.StatusUnauthorized},
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
	f := newFakeCognito(t)
	h := denyGate(f)
	for _, path := range []string{"/admin/report", "/api/auth/delete", "/api/auth/anything-else"} {
		rec := authDo(t, h, http.MethodGet, path, "", withBearer(f.idToken("user-a", nil)))
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
