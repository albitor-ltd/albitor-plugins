package shell

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"webapp/internal/apikit"
	"webapp/internal/identity"
)

// publicTestSetup is a pool the test can mint tokens for, with the posture left private.
func publicTestSetup(t *testing.T) (signIn, func(string, map[string]any) string) {
	f := newFakeCognito(t)
	return signIn{auth: f.auth()}, f.idToken
}

// publicMux serves the routes the tests name: two public reads and one public write that say who
// they saw, and one private read.
func publicMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		apikit.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	seen := func(w http.ResponseWriter, r *http.Request) {
		who := "nobody"
		if id, ok := identity.FromContext(r.Context()); ok {
			who = id.Sub
		}
		if r.Method == http.MethodPost {
			if _, err := io.ReadAll(r.Body); err != nil {
				apikit.WriteError(w, http.StatusRequestEntityTooLarge, "too large")
				return
			}
		}
		apikit.WriteJSON(w, http.StatusOK, map[string]string{"who": who})
	}
	for _, p := range []string{"/api/listings", "/api/listings/{id}", "/api/enquiries", "/api/saved", "/api/docs/", "/api/admin/stats"} {
		mux.HandleFunc(p, seen)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		apikit.WriteError(w, http.StatusNotFound, "not found")
	})
	return mux
}

const testPublicRoutes = "/listings, /listings/*, GET /api/listings, GET /api/listings/{id}, POST /api/enquiries, GET /api/docs/*"

func publicGate(t *testing.T, routes string, bodyMax string) (http.Handler, func(string, map[string]any) string) {
	t.Helper()
	pol, err := parsePublicPolicy("public", routes, bodyMax)
	if err != nil {
		t.Fatal(err)
	}
	setup, mint := publicTestSetup(t)
	setup.public = pol
	return newIdentityGate(publicMux(), setup, identityTestLogger()), mint
}

func TestPublicRoutesAreOpenSignedOutAndEverythingElseStaysClosed(t *testing.T) {
	h, _ := publicGate(t, testPublicRoutes, "100")
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/listings", http.StatusOK},
		{http.MethodHead, "/api/listings", http.StatusOK},
		{http.MethodGet, "/api/listings/abc", http.StatusOK},
		{http.MethodPost, "/api/enquiries", http.StatusOK},
		{http.MethodGet, "/api/docs/a", http.StatusOK},
		{http.MethodGet, "/api/docs/a/b", http.StatusOK},
		// An encoded slash, or a dot segment in a wildcard's tail, is never public.
		{http.MethodGet, "/api/docs/a%2Fb", http.StatusUnauthorized},
		{http.MethodGet, "/api/docs/a%2fb", http.StatusUnauthorized},
		{http.MethodGet, "/api/docs/../admin/stats", http.StatusUnauthorized},
		{http.MethodGet, "/api/docs/./a", http.StatusUnauthorized},
		// Not listed: the method, the shape, a sibling, a private route.
		{http.MethodPost, "/api/listings", http.StatusUnauthorized},
		{http.MethodDelete, "/api/listings/abc", http.StatusUnauthorized},
		{http.MethodGet, "/api/listings/abc/extra", http.StatusNotFound}, // nothing serves it
		{http.MethodGet, "/api/enquiries", http.StatusUnauthorized},
		{http.MethodGet, "/api/saved", http.StatusUnauthorized},
		{http.MethodGet, "/api/admin/stats", http.StatusUnauthorized},
		// A wildcard needs at least one more segment, and an empty segment matches nothing.
		{http.MethodGet, "/api/docs/", http.StatusUnauthorized},
		{http.MethodGet, "/api/listings/", http.StatusNotFound}, // nothing serves it
		// A web entry opens no API route.
		{http.MethodGet, "/listings", http.StatusNotFound},
	} {
		rec := authDo(t, h, tc.method, tc.path, "", nil)
		if rec.Code != tc.want {
			t.Errorf("signed out, %s %s = %d, want %d: %s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}

func TestAPublicRouteTakesIdentityAsOptional(t *testing.T) {
	h, mint := publicGate(t, testPublicRoutes, "")
	who := func(rec *httptest.ResponseRecorder) string {
		var out map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return out["who"]
	}

	// No token: anonymous, never a fixed owner.
	rec := authDo(t, h, http.MethodGet, "/api/listings", "", nil)
	if rec.Code != http.StatusOK || who(rec) != "nobody" {
		t.Fatalf("signed out = %d %s, want 200 with no identity", rec.Code, rec.Body.String())
	}
	// A valid token is verified and used.
	rec = authDo(t, h, http.MethodGet, "/api/listings", "", withBearer(mint("user-a", nil)))
	if rec.Code != http.StatusOK || who(rec) != "user-a" {
		t.Fatalf("signed in = %d %s, want 200 for user-a", rec.Code, rec.Body.String())
	}
	// A token that is present but bad is a 401, never a downgrade to anonymous.
	for name, edit := range map[string]func(*http.Request){
		"an expired bearer":      withBearer(mint("user-a", map[string]any{"exp": time.Now().Add(-time.Hour).Unix()})),
		"another client's token": withBearer(mint("user-a", map[string]any{"aud": "other-client"})),
		"garbage":                withBearer("not-a-token"),
		"a non-bearer header":    func(r *http.Request) { r.Header.Set("Authorization", "Basic dXNlcjpwYXNz") },
		"a garbage session":      func(r *http.Request) { r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "not-a-token"}) },
	} {
		if rec := authDo(t, h, http.MethodGet, "/api/listings", "", edit); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s on a public route = %d, want 401: %s", name, rec.Code, rec.Body.String())
		}
	}
}

func TestAPublicWriteIsCappedAndAPrivateWriteIsNot(t *testing.T) {
	h, mint := publicGate(t, testPublicRoutes, "20")
	small, big := strings.Repeat("x", 20), strings.Repeat("x", 21)
	if rec := authDo(t, h, http.MethodPost, "/api/enquiries", small, nil); rec.Code != http.StatusOK {
		t.Errorf("a body at the cap = %d, want 200", rec.Code)
	}
	if rec := authDo(t, h, http.MethodPost, "/api/enquiries", big, nil); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("a body over the cap = %d, want 413", rec.Code)
	}
	// The cap holds for a signed-in caller too, and for a body that does not declare its length.
	if rec := authDo(t, h, http.MethodPost, "/api/enquiries", big, withBearer(mint("user-a", nil))); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("a signed-in body over the cap = %d, want 413", rec.Code)
	}
	chunked := func(r *http.Request) { r.ContentLength = -1 }
	if rec := authDo(t, h, http.MethodPost, "/api/enquiries", big, chunked); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("an undeclared body over the cap = %d, want the handler's 413", rec.Code)
	}
}

func TestAPublicPolicyThatIsMalformedFailsClosed(t *testing.T) {
	for name, tc := range map[string]struct{ posture, routes, max string }{
		"public with no list":  {"public", "", ""},
		"a blank entry":        {"public", "GET /api/a,,GET /api/b", ""},
		"an unknown method":    {"public", "FETCH /api/a", ""},
		"no method":            {"public", "/api/a b", ""},
		"a path outside /api/": {"public", "GET /other", ""},
		"a root wildcard":      {"public", "GET /api/*", ""},
		"a mid-path wildcard":  {"public", "GET /api/a/*/b", ""},
		"a partial wildcard":   {"public", "GET /api/a*", ""},
		"an empty segment":     {"public", "GET /api/a//b", ""},
		"a dot-dot segment":    {"public", "GET /api/a/../b", ""},
		"an empty parameter":   {"public", "GET /api/a/{}", ""},
		"a bad character":      {"public", "GET /api/a?b", ""},
		"a zero cap":           {"public", "GET /api/a", "0"},
		"a non-numeric cap":    {"public", "GET /api/a", "10k"},
		"an unknown posture":   {"open", "GET /api/a", ""},
		"private with a list":  {"private", "GET /api/a", ""},
		"too many entries":     {"public", strings.TrimSuffix(strings.Repeat("GET /api/a,", maxPublicRoutes+1), ","), ""},
	} {
		if _, err := parsePublicPolicy(tc.posture, tc.routes, tc.max); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	// Private and unset are closed: nothing is public.
	for _, posture := range []string{"", "private"} {
		p, err := parsePublicPolicy(posture, "", "")
		if err != nil || p.public {
			t.Errorf("posture %q = %+v, %v, want private", posture, p, err)
		}
	}
}

func TestPrivatePostureOpensNothing(t *testing.T) {
	setup, _ := publicTestSetup(t)
	h := newIdentityGate(publicMux(), setup, identityTestLogger())
	for _, path := range []string{"/api/listings", "/api/listings/abc"} {
		if rec := authDo(t, h, http.MethodGet, path, "", nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("private, signed out, GET %s = %d, want 401", path, rec.Code)
		}
	}
}

func TestTheBodyCapDefaultsTo16384(t *testing.T) {
	p, err := parsePublicPolicy("public", "POST /api/a", "")
	if err != nil || p.bodyMax != 16384 {
		t.Fatalf("default cap = %d, %v, want 16384", p.bodyMax, err)
	}
}
