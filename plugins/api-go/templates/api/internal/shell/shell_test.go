package shell

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"webapp/internal/apikit"
	"webapp/internal/localauth"
)

func quietLogger() *slog.Logger { return slog.New(slog.NewJSONHandler(io.Discard, nil)) }

func serve(t *testing.T, h http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func mustBuild(t *testing.T, chain []Middleware, routes []Route) http.Handler {
	t.Helper()
	h, err := build(quietLogger(), chain, routes)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return h
}

func TestHealthAnswersOKOnGetOnly(t *testing.T) {
	h := mustBuild(t, nil, nil)

	rec := serve(t, h, http.MethodGet, "/api/health")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("GET /api/health = %d %s", rec.Code, rec.Body.String())
	}
	if rec := serve(t, h, http.MethodPost, "/api/health"); rec.Code != http.StatusNotFound {
		t.Fatalf("POST /api/health = %d, want 404", rec.Code)
	}
}

func TestUnmatchedPathIsAJSON404(t *testing.T) {
	rec := serve(t, mustBuild(t, nil, nil), http.MethodGet, "/api/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] == "" {
		t.Fatalf("body = %q, want {\"error\":...}", rec.Body.String())
	}
}

func TestRoutesAreServedAndTheFallbackStillCatchesTheRest(t *testing.T) {
	h := mustBuild(t, nil, []Route{
		func(r *http.ServeMux) {
			r.HandleFunc("GET /api/widgets", func(w http.ResponseWriter, _ *http.Request) {
				apikit.WriteJSON(w, http.StatusOK, apikit.Page[string]{Items: []string{"a"}})
			})
		},
		func(r *http.ServeMux) {
			r.HandleFunc("GET /api/gadgets", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		},
	})
	if rec := serve(t, h, http.MethodGet, "/api/widgets"); rec.Code != http.StatusOK {
		t.Fatalf("/api/widgets = %d", rec.Code)
	}
	if rec := serve(t, h, http.MethodGet, "/api/gadgets"); rec.Code != http.StatusNoContent {
		t.Fatalf("/api/gadgets = %d", rec.Code)
	}
	if rec := serve(t, h, http.MethodGet, "/api/other"); rec.Code != http.StatusNotFound {
		t.Fatalf("/api/other = %d, want 404", rec.Code)
	}
}

// The first entry of the slot is the outermost middleware, and the chain runs
// inside request logging.
func TestMiddlewareRunsInSlotOrderOutermostFirst(t *testing.T) {
	var trace []string
	tag := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				trace = append(trace, name+">")
				next.ServeHTTP(w, r)
				trace = append(trace, "<"+name)
			})
		}
	}
	h := mustBuild(t, []Middleware{tag("first"), tag("second")}, nil)
	serve(t, h, http.MethodGet, "/api/health")

	if got := strings.Join(trace, " "); got != "first> second> <second <first" {
		t.Fatalf("trace = %q", got)
	}
}

// A middleware that refuses a request (an auth check) is still logged and
// correlated, because request logging is outside the chain.
func TestARefusedRequestStillGetsACorrelationIDAndALogLine(t *testing.T) {
	var sink bytes.Buffer
	refuse := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			apikit.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		})
	}
	h, err := build(slog.New(slog.NewJSONHandler(&sink, nil)), []Middleware{refuse}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := serve(t, h, http.MethodGet, "/api/anything")

	if rec.Code != http.StatusUnauthorized || rec.Header().Get(apikit.CorrelationIDHeader) == "" {
		t.Fatalf("status = %d, correlation id = %q", rec.Code, rec.Header().Get(apikit.CorrelationIDHeader))
	}
	if !strings.Contains(sink.String(), `"status":401`) {
		t.Fatalf("no log line for the refused request: %q", sink.String())
	}
}

func TestNilMiddlewareAndNilRouteAreStartErrors(t *testing.T) {
	if _, err := build(quietLogger(), []Middleware{nil}, nil); err == nil {
		t.Fatal("a nil middleware started")
	}
	if _, err := build(quietLogger(), nil, []Route{nil}); err == nil {
		t.Fatal("a nil route registration started")
	}
}

// The composed router: with no plugin contributions it still serves health and
// the JSON 404. This runs against the generated router.go.
func TestComposedHandlerServesHealth(t *testing.T) {
	h, err := Handler(quietLogger())
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	if rec := serve(t, h, http.MethodGet, "/api/health"); rec.Code != http.StatusOK {
		t.Fatalf("GET /api/health = %d", rec.Code)
	}
}

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

// The Go side of the local run contract: ALBITOR_LOCAL=1 boots with no AWS and
// no configuration, and ALBITOR_LOCAL_AUTH=1 adds the test issuer.
func TestBootInALocalRun(t *testing.T) {
	plain, err := Boot(env(map[string]string{localauth.LocalModeEnv: "1"}), quietLogger())
	if err != nil {
		t.Fatalf("Boot: %v", err)
	}
	if rec := serve(t, plain, http.MethodGet, "/api/health"); rec.Code != http.StatusOK {
		t.Fatalf("health = %d", rec.Code)
	}
	// Not 404: a composed middleware (sign-in) may answer 401 for the unknown path.
	if rec := serve(t, plain, http.MethodGet, localauth.Path+"/identities"); rec.Code == http.StatusOK {
		t.Fatalf("without ALBITOR_LOCAL_AUTH the issuer answered %d", rec.Code)
	}

	withAuth, err := Boot(env(map[string]string{localauth.LocalModeEnv: "1", localauth.EnabledEnv: "1"}), quietLogger())
	if err != nil {
		t.Fatalf("Boot with the issuer: %v", err)
	}
	if rec := serve(t, withAuth, http.MethodGet, localauth.Path+"/identities"); rec.Code != http.StatusOK {
		t.Fatalf("issuer identities = %d", rec.Code)
	}
	if rec := serve(t, withAuth, http.MethodGet, "/api/health"); rec.Code != http.StatusOK {
		t.Fatalf("health beside the issuer = %d", rec.Code)
	}
}

// The issuer must never start outside a local run: Boot fails rather than
// serving test tokens from a deployed function.
func TestBootRefusesTheIssuerOnADeployedFunction(t *testing.T) {
	_, err := Boot(env(map[string]string{
		localauth.LocalModeEnv: "1", localauth.EnabledEnv: "1", "AWS_LAMBDA_FUNCTION_NAME": "app-api",
	}), quietLogger())
	if err == nil {
		t.Fatal("Boot started the issuer on a deployed function")
	}
}

// Each guard refuses the issuer on its own, through Boot.
func TestBootRefusesTheIssuerWhenAnyGateFails(t *testing.T) {
	cases := map[string]map[string]string{
		"ALBITOR_LOCAL unset":   {localauth.EnabledEnv: "1"},
		"ALBITOR_LOCAL not 1":   {localauth.EnabledEnv: "1", localauth.LocalModeEnv: "true"},
		"deployed function":     {localauth.EnabledEnv: "1", localauth.LocalModeEnv: "1", "AWS_LAMBDA_FUNCTION_NAME": "f"},
		"AUTH_ISSUER set":       {localauth.EnabledEnv: "1", localauth.LocalModeEnv: "1", "AUTH_ISSUER": "https://x"},
		"AUTH_USER_POOL_ID set": {localauth.EnabledEnv: "1", localauth.LocalModeEnv: "1", "AUTH_USER_POOL_ID": "p"},
		"AUTH_CLIENT_ID set":    {localauth.EnabledEnv: "1", localauth.LocalModeEnv: "1", "AUTH_CLIENT_ID": "c"},
	}
	for name, vars := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Boot(env(vars), quietLogger()); err == nil {
				t.Fatal("Boot started the issuer")
			}
		})
	}
}

// The issuer mints a token for anyone who can reach it, so when it runs the
// server binds loopback only; otherwise it listens on every interface.
func TestListenAddrIsLoopbackOnlyWhileTheIssuerRuns(t *testing.T) {
	cases := []struct {
		vars map[string]string
		want string
	}{
		{map[string]string{}, ":8080"},
		{map[string]string{"PORT": "9000"}, ":9000"},
		{map[string]string{localauth.LocalModeEnv: "1"}, ":8080"},
		{map[string]string{localauth.LocalModeEnv: "1", localauth.EnabledEnv: "1"}, "127.0.0.1:8080"},
		{map[string]string{localauth.LocalModeEnv: "1", localauth.EnabledEnv: "1", "PORT": "9000"}, "127.0.0.1:9000"},
	}
	for _, tc := range cases {
		if got := ListenAddr(env(tc.vars)); got != tc.want {
			t.Errorf("ListenAddr(%v) = %q, want %q", tc.vars, got, tc.want)
		}
	}
}
