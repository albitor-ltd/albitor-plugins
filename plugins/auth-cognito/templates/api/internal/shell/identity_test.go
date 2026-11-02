package shell

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"webapp/internal/apikit"
	"webapp/internal/identity"
	"webapp/internal/localauth"
)

// These tests hold the identity seam to its contract: /api/health and the
// sign-in calls open, every other /api/ route refused without a verified
// token, the owner taken from the token's sub, and nothing let through when the
// pool cannot be reached. A fake Cognito serves the JWKS and the public calls.

const (
	testClientID = "client-under-test"
	testKid      = "key-1"
)

type fakeCognito struct {
	t      *testing.T
	key    *rsa.PrivateKey
	issuer string

	mu      sync.Mutex
	calls   []string
	answers map[string]func(in map[string]any) (int, any)
}

func newFakeCognito(t *testing.T) (*fakeCognito, *httptest.Server) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeCognito{t: t, key: key, answers: map[string]func(map[string]any) (int, any){}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	f.issuer = srv.URL + "/pool"
	return f, srv
}

func (f *fakeCognito) serve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/pool/.well-known/jwks.json" {
		pub := f.key.PublicKey
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": testKid, "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}}})
		return
	}
	op := strings.TrimPrefix(r.Header.Get("X-Amz-Target"), "AWSCognitoIdentityProviderService.")
	var in map[string]any
	_ = json.NewDecoder(r.Body).Decode(&in)
	f.mu.Lock()
	f.calls = append(f.calls, op)
	answer := f.answers[op]
	f.mu.Unlock()
	if answer == nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"__type":"InternalErrorException","message":"unexpected call"}`)
		return
	}
	status, body := answer(in)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (f *fakeCognito) answer(op string, fn func(in map[string]any) (int, any)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answers[op] = fn
}

func (f *fakeCognito) called(op string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if c == op {
			return true
		}
	}
	return false
}

// token signs claims over the defaults of a valid ID token for sub.
func (f *fakeCognito) token(sub string, override map[string]any) string {
	return signToken(f.t, f.key, testKid, "RS256", f.claims(sub, override))
}

func (f *fakeCognito) claims(sub string, override map[string]any) map[string]any {
	now := time.Now()
	claims := map[string]any{
		"iss": f.issuer, "sub": sub, "aud": testClientID, "token_use": "id",
		"email": sub + "@example.invalid", "iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
	}
	for k, v := range override {
		claims[k] = v
	}
	return claims
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid, alg string, claims map[string]any) string {
	t.Helper()
	enc := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(b)
	}
	signing := enc(map[string]string{"alg": alg, "kid": kid}) + "." + enc(claims)
	digest := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func identityTestLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// composeIdentity assembles what the generated router does: the identity
// routes on a mux with a health check, one user-owned route that echoes its
// owner and the catch-all JSON 404, behind the identity middleware.
func composeIdentity(setup signIn) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		apikit.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		apikit.WriteJSON(w, http.StatusOK, map[string]string{"owner": identity.Owner(r)})
	})
	registerIdentityRoutes(mux, setup, identityTestLogger())
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		apikit.WriteError(w, http.StatusNotFound, "not found")
	})
	return newIdentityGate(mux, setup, identityTestLogger())
}

func gateFor(f *fakeCognito, srv *httptest.Server) http.Handler {
	return composeIdentity(signIn{auth: &cognitoAuth{
		clientID: testClientID,
		endpoint: srv.URL + "/",
		client:   srv.Client(),
		verifier: newCognitoVerifier(f.issuer, testClientID, f.issuer+"/.well-known/jwks.json", srv.Client()),
	}})
}

func authDo(t *testing.T, h http.Handler, method, path, body string, edit func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	if edit != nil {
		edit(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func withBearer(token string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
}

func withCookies(cookies ...*http.Cookie) func(*http.Request) {
	return func(r *http.Request) {
		for _, c := range cookies {
			r.AddCookie(c)
		}
	}
}

func authDecode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("response is not JSON: %q", rec.Body.String())
	}
	return out
}

func TestHealthAndUnknownRoutesNeedNoSignIn(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)

	if rec := authDo(t, h, http.MethodGet, "/api/health", "", nil); rec.Code != http.StatusOK {
		t.Fatalf("GET /api/health = %d, want 200", rec.Code)
	}
	// The deploy's smoke check wants an unknown /api/ path to be a JSON 404.
	if rec := authDo(t, h, http.MethodGet, "/api/no-such-route", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("GET /api/no-such-route = %d, want 404", rec.Code)
	}
}

func TestAProtectedRouteRefusesARequestWithoutAVerifiedToken(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]func(*http.Request){
		"no token":            nil,
		"not a token":         withBearer("not-a-token"),
		"signed by another":   withBearer(signToken(t, other, testKid, "RS256", f.claims("intruder", nil))),
		"alg none":            withBearer(signToken(t, f.key, testKid, "none", f.claims("intruder", nil))),
		"another pool":        withBearer(f.token("intruder", map[string]any{"iss": "https://cognito-idp.eu-west-2.amazonaws.com/other"})),
		"another client":      withBearer(f.token("intruder", map[string]any{"aud": "other-client"})),
		"refresh token use":   withBearer(f.token("intruder", map[string]any{"token_use": "refresh"})),
		"expired":             withBearer(f.token("intruder", map[string]any{"exp": time.Now().Add(-time.Minute).Unix()})),
		"unknown kid":         withBearer(signToken(t, f.key, "key-2", "RS256", f.claims("intruder", nil))),
		"basic auth":          func(r *http.Request) { r.SetBasicAuth("someone", "anything") },
		"expired, no refresh": withCookies(&http.Cookie{Name: sessionCookie, Value: f.token("intruder", map[string]any{"exp": time.Now().Add(-time.Minute).Unix()})}),
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			rec := authDo(t, h, http.MethodGet, "/api/items", "", edit)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("GET /api/items = %d, want 401: %s", rec.Code, rec.Body.String())
			}
			if got := authDecode(t, rec)["code"]; got != "unauthenticated" {
				t.Fatalf("code = %v, want unauthenticated", got)
			}
		})
	}
}

func TestTheOwnerIsTheVerifiedSub(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)

	for name, edit := range map[string]func(*http.Request){
		"bearer id token":     withBearer(f.token("user-a", nil)),
		"bearer access token": withBearer(f.token("user-a", map[string]any{"token_use": "access", "aud": nil, "client_id": testClientID})),
		"session cookie":      withCookies(&http.Cookie{Name: sessionCookie, Value: f.token("user-a", nil)}),
	} {
		t.Run(name, func(t *testing.T) {
			rec := authDo(t, h, http.MethodGet, "/api/items", "", edit)
			if rec.Code != http.StatusOK {
				t.Fatalf("GET /api/items = %d: %s", rec.Code, rec.Body.String())
			}
			if got := authDecode(t, rec)["owner"]; got != "user-a" {
				t.Fatalf("owner = %v, want user-a", got)
			}
		})
	}
}

func TestMeAnswersTheSignedInPersonWithTheirGroups(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)

	if rec := authDo(t, h, http.MethodGet, "/api/me", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/me signed out = %d, want 401", rec.Code)
	}
	rec := authDo(t, h, http.MethodGet, "/api/me", "", withBearer(f.token("user-a", map[string]any{"cognito:groups": []string{"admin"}})))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/me = %d", rec.Code)
	}
	me := authDecode(t, rec)
	groups, _ := me["groups"].([]any)
	if me["sub"] != "user-a" || me["email"] != "user-a@example.invalid" || len(groups) != 1 || groups[0] != "admin" {
		t.Fatalf("GET /api/me = %v", me)
	}
	if rec := authDo(t, h, http.MethodPost, "/api/me", "", withBearer(f.token("user-a", nil))); rec.Code != http.StatusNotFound {
		t.Fatalf("POST /api/me = %d, want 404", rec.Code)
	}
}

func TestAnExpiredSessionIsRefreshedFromTheRefreshCookie(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	f.answer("InitiateAuth", func(in map[string]any) (int, any) {
		params, _ := in["AuthParameters"].(map[string]any)
		if in["AuthFlow"] != "REFRESH_TOKEN_AUTH" || params["REFRESH_TOKEN"] != "refresh-a" {
			return http.StatusBadRequest, map[string]string{"__type": "NotAuthorizedException"}
		}
		return http.StatusOK, map[string]any{"AuthenticationResult": map[string]string{"IdToken": f.token("user-a", nil)}}
	})

	expired := f.token("user-a", map[string]any{"exp": time.Now().Add(-time.Minute).Unix()})
	rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(
		&http.Cookie{Name: sessionCookie, Value: expired},
		&http.Cookie{Name: refreshCookie, Value: "refresh-a"},
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/items = %d: %s", rec.Code, rec.Body.String())
	}
	if got := authDecode(t, rec)["owner"]; got != "user-a" {
		t.Fatalf("owner = %v", got)
	}
	session := responseCookie(rec, sessionCookie)
	if session == nil || session.Value == "" || session.Value == expired {
		t.Fatalf("the refreshed session was not set: %v", rec.Result().Cookies())
	}

	// A refresh token the pool refuses ends the session.
	rec = authDo(t, h, http.MethodGet, "/api/items", "", withCookies(
		&http.Cookie{Name: sessionCookie, Value: expired},
		&http.Cookie{Name: refreshCookie, Value: "revoked"},
	))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/items with a revoked refresh = %d, want 401", rec.Code)
	}
	if c := responseCookie(rec, sessionCookie); c == nil || c.MaxAge >= 0 {
		t.Fatalf("the dead session was not cleared: %v", rec.Result().Cookies())
	}
}

func responseCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestAMisconfiguredPoolFailsClosed(t *testing.T) {
	h := composeIdentity(signIn{err: errTestConfig})

	if rec := authDo(t, h, http.MethodGet, "/api/health", "", nil); rec.Code != http.StatusOK {
		t.Fatalf("GET /api/health = %d, want 200 even with no pool", rec.Code)
	}
	for _, path := range []string{"/api/items", "/api/me"} {
		if rec := authDo(t, h, http.MethodGet, path, "", nil); rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("GET %s = %d, want 503", path, rec.Code)
		}
	}
}

var errTestConfig = io.ErrUnexpectedEOF

func identityTestEnv(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func TestSignInFromEnv(t *testing.T) {
	pool := map[string]string{"AUTH_USER_POOL_ID": "eu-west-2_abc", "AUTH_CLIENT_ID": "client", "AWS_REGION": "eu-west-2"}

	s := signInFromEnv(identityTestEnv(pool))
	if s.err != nil || s.auth == nil || s.anonymous || s.auth.local {
		t.Fatalf("a named pool = %+v, want the pool", s)
	}
	if got := s.auth.verifier.issuer; got != "https://cognito-idp.eu-west-2.amazonaws.com/eu-west-2_abc" {
		t.Fatalf("issuer = %q", got)
	}

	for name, vars := range map[string]map[string]string{
		"a pool with no client":      {"AUTH_USER_POOL_ID": "eu-west-2_abc", "AWS_REGION": "eu-west-2"},
		"a client with no pool":      {"AUTH_CLIENT_ID": "client", "AWS_REGION": "eu-west-2"},
		"a pool with no region":      {"AUTH_USER_POOL_ID": "eu-west-2_abc", "AUTH_CLIENT_ID": "client"},
		"an unsafe pool id":          {"AUTH_USER_POOL_ID": "eu-west-2_abc/../x", "AUTH_CLIENT_ID": "client", "AWS_REGION": "eu-west-2"},
		"a deployed function":        {"AWS_LAMBDA_FUNCTION_NAME": "app-dev-api"},
		"no pool and not local":      {},
		"the issuer switch alone":    {"ALBITOR_LOCAL_AUTH": "1"},
		"local mode that is not 1":   {"ALBITOR_LOCAL": "true"},
		"local mode on a deployment": {"ALBITOR_LOCAL": "1", "AWS_LAMBDA_FUNCTION_NAME": "app-dev-api"},
	} {
		t.Run(name, func(t *testing.T) {
			if s := signInFromEnv(identityTestEnv(vars)); s.err == nil || s.auth != nil || s.anonymous {
				t.Fatalf("%s = %+v, want a closed failure", name, s)
			}
		})
	}

	if s := signInFromEnv(identityTestEnv(map[string]string{"ALBITOR_LOCAL": "1"})); !s.anonymous || s.err != nil || s.auth != nil {
		t.Fatalf("a local run with no issuer = %+v, want anonymous", s)
	}
}

func TestWithoutAPoolEveryRequestIsAnonymous(t *testing.T) {
	h := composeIdentity(signIn{anonymous: true})

	rec := authDo(t, h, http.MethodGet, "/api/items", "", nil)
	if got := authDecode(t, rec)["owner"]; got != identity.AnonymousOwner {
		t.Fatalf("owner = %v, want %s", got, identity.AnonymousOwner)
	}
	if rec := authDo(t, h, http.MethodGet, "/api/me", "", nil); authDecode(t, rec)["sub"] != identity.AnonymousOwner {
		t.Fatalf("GET /api/me = %s", rec.Body.String())
	}
}

// The local issuer is trusted in a local run with ALBITOR_LOCAL_AUTH=1, and
// never where a pool is named or the function is deployed, whatever else is set.
func TestTheLocalIssuerIsTrustedOnlyInALocalRun(t *testing.T) {
	local := map[string]string{"ALBITOR_LOCAL": "1", "ALBITOR_LOCAL_AUTH": "1", "PORT": "8080"}

	s := signInFromEnv(identityTestEnv(local))
	if s.err != nil || s.auth == nil || !s.auth.local {
		t.Fatalf("a local run with the local issuer = %+v, want it trusted", s)
	}
	if s.auth.verifier.issuer != "http://localhost:8080"+localauth.Path || s.auth.clientID != localauth.ClientID {
		t.Fatalf("local verifier = %q for %q, want the local issuer's URL and client", s.auth.verifier.issuer, s.auth.clientID)
	}

	with := func(extra map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range local {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	for name, vars := range map[string]map[string]string{
		"on a deployed function": with(map[string]string{"AWS_LAMBDA_FUNCTION_NAME": "app-dev-api"}),
		"beside a named pool":    with(map[string]string{"AUTH_USER_POOL_ID": "eu-west-2_abc", "AUTH_CLIENT_ID": "c", "AWS_REGION": "eu-west-2"}),
		"beside a client id":     with(map[string]string{"AUTH_CLIENT_ID": "c"}),
	} {
		t.Run(name, func(t *testing.T) {
			if s := signInFromEnv(identityTestEnv(vars)); s.auth != nil && s.auth.local {
				t.Fatal("the local issuer was trusted where a pool or a deployed function decides")
			}
		})
	}
	for name, vars := range map[string]map[string]string{
		"without local mode": {"ALBITOR_LOCAL_AUTH": "1"},
		"local mode not 1":   {"ALBITOR_LOCAL": "true", "ALBITOR_LOCAL_AUTH": "1"},
		"issuer not 1":       {"ALBITOR_LOCAL": "1", "ALBITOR_LOCAL_AUTH": "true"},
	} {
		t.Run(name, func(t *testing.T) {
			if s := signInFromEnv(identityTestEnv(vars)); s.auth != nil {
				t.Fatal("the local issuer was trusted without both local switches")
			}
		})
	}
}

// startLocalRun serves the composed identity seam behind the local issuer
// on a real port, as `ALBITOR_LOCAL=1 ALBITOR_LOCAL_AUTH=1 go run .` does, so
// the verifier fetches the issuer's JWKS over HTTP.
func startLocalRun(t *testing.T) (*httptest.Server, *localauth.Issuer) {
	t.Helper()
	srv := httptest.NewUnstartedServer(nil)
	port := strings.TrimPrefix(srv.Listener.Addr().String(), "127.0.0.1:")
	t.Setenv("ALBITOR_LOCAL", "1")
	t.Setenv("ALBITOR_LOCAL_AUTH", "1")
	t.Setenv("PORT", port)
	issuer, err := localauth.Open(os.Getenv)
	if err != nil || issuer == nil {
		t.Fatalf("localauth.Open = %v, %v", issuer, err)
	}
	setup := signInFromEnv(os.Getenv)
	if setup.err != nil || setup.auth == nil || !setup.auth.local {
		t.Fatalf("signInFromEnv in a local run = %+v", setup)
	}
	srv.Config.Handler = issuer.Wrap(composeIdentity(setup))
	srv.Start()
	t.Cleanup(srv.Close)
	return srv, issuer
}

// A pool's verifier refuses the local issuer's tokens: another issuer, and a
// key the pool never published.
func TestAPoolRefusesTheLocalIssuersTokens(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	_, issuer := startLocalRun(t)
	tokens, err := issuer.Mint("alice")
	if err != nil {
		t.Fatal(err)
	}
	for name, token := range map[string]string{"id token": tokens.IDToken, "access token": tokens.AccessToken} {
		if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(token)); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s from the local issuer = %d, want 401", name, rec.Code)
		}
	}
	// Control: the pool's own token passes, so the refusal is the verifier's.
	if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(f.token("user-a", nil))); rec.Code != http.StatusOK {
		t.Fatalf("the pool's own token = %d, want 200", rec.Code)
	}
}
