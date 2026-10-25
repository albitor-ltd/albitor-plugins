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
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"webapp/internal/apikit"
	"webapp/internal/identity"
	"webapp/internal/localauth"
)

// These tests hold the identity seam to its contract: /api/health and the sign-in routes open,
// every other /api/ route refused without a verified token, the owner taken from the token's sub,
// and nothing let through when the pool is not configured or cannot be reached. A fake Cognito
// serves the JWKS, the hosted token and revoke endpoints and the user pool's DeleteUser.

const (
	testClientID    = "client-under-test"
	testKid         = "key-1"
	testRedirectURI = "https://app.test/api/auth/callback"
	testLogoutURL   = "https://pool.auth.test/logout?client_id=client-under-test&logout_uri=https%3A%2F%2Fapp.test%2F"
)

// fakeCall is one request the fake pool received.
type fakeCall struct {
	path   string
	header http.Header
	form   url.Values // a form post (the token and revoke endpoints)
	json   map[string]any
}

type fakeCognito struct {
	t      *testing.T
	key    *rsa.PrivateKey
	issuer string
	srv    *httptest.Server

	mu    sync.Mutex
	calls []fakeCall
	// token answers POST /oauth2/token; revoke POST /oauth2/revoke; deleteUser the DeleteUser call.
	token      func(form url.Values) (int, any)
	revoke     func(form url.Values) int
	deleteUser func(in map[string]any) (int, any)
}

func newFakeCognito(t *testing.T) *fakeCognito {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeCognito{t: t, key: key}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	f.issuer = f.srv.URL + "/pool"
	return f
}

func (f *fakeCognito) serve(w http.ResponseWriter, r *http.Request) {
	call := fakeCall{path: r.URL.Path, header: r.Header.Clone()}
	if r.URL.Path == "/pool/.well-known/jwks.json" {
		pub := f.key.PublicKey
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": testKid, "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}}})
		return
	}
	body, _ := io.ReadAll(r.Body)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		call.form, _ = url.ParseQuery(string(body))
	} else {
		_ = json.Unmarshal(body, &call.json)
		call.path += "#" + strings.TrimPrefix(r.Header.Get("X-Amz-Target"), "AWSCognitoIdentityProviderService.")
	}
	f.mu.Lock()
	f.calls = append(f.calls, call)
	tokenFn, revokeFn, deleteFn := f.token, f.revoke, f.deleteUser
	f.mu.Unlock()

	switch {
	case r.URL.Path == "/oauth2/token" && tokenFn != nil:
		status, out := tokenFn(call.form)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(out)
	case r.URL.Path == "/oauth2/revoke" && revokeFn != nil:
		w.WriteHeader(revokeFn(call.form))
	case r.URL.Path == "/" && strings.HasSuffix(call.path, "#DeleteUser") && deleteFn != nil:
		status, out := deleteFn(call.json)
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(out)
	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"unexpected_call"}`)
	}
}

// callsTo returns the calls whose path is path.
func (f *fakeCognito) callsTo(path string) []fakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []fakeCall
	for _, c := range f.calls {
		if c.path == path {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeCognito) answerToken(fn func(form url.Values) (int, any)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.token = fn
}

// claims returns the defaults of a valid ID token for sub, less what override replaces.
func (f *fakeCognito) claims(sub string, override map[string]any) map[string]any {
	now := time.Now()
	claims := map[string]any{
		"iss": f.issuer, "sub": sub, "aud": testClientID, "token_use": "id",
		"email": sub + "@example.invalid", "email_verified": true,
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
	}
	for k, v := range override {
		claims[k] = v
	}
	return claims
}

// token signs an ID token for sub with override applied. A nil override value deletes the claim.
func (f *fakeCognito) idToken(sub string, override map[string]any) string {
	c := f.claims(sub, override)
	for k, v := range c {
		if v == nil {
			delete(c, k)
		}
	}
	return signToken(f.t, f.key, testKid, "RS256", c)
}

// accessToken signs an access token for sub carrying scope.
func (f *fakeCognito) accessToken(sub, scope string) string {
	return signToken(f.t, f.key, testKid, "RS256", map[string]any{
		"iss": f.issuer, "sub": sub, "client_id": testClientID, "token_use": "access", "scope": scope,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	})
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

// auth builds the hosted setup around the fake, as hostedConfigFromEnv would for a real pool.
func (f *fakeCognito) auth() *hostedAuth {
	return &hostedAuth{
		clientID: testClientID,
		loginURL: f.srv.URL + "/oauth2/authorize?client_id=" + testClientID +
			"&response_type=code&scope=openid%20email%20aws.cognito.signin.user.admin&redirect_uri=https%3A%2F%2Fapp.test%2Fapi%2Fauth%2Fcallback",
		logoutURL:   testLogoutURL,
		redirectURI: testRedirectURI,
		tokenURL:    f.srv.URL + "/oauth2/token",
		revokeURL:   f.srv.URL + "/oauth2/revoke",
		endpoint:    f.srv.URL + "/",
		client:      f.srv.Client(),
		verifier:    newCognitoVerifier(f.issuer, testClientID, f.issuer+"/.well-known/jwks.json", f.srv.Client()),
		now:         time.Now,
	}
}

func identityTestLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// composeIdentity assembles what the generated router does: the identity routes on a mux with a
// health check, one user-owned route that echoes its owner and the catch-all JSON 404, behind the
// identity middleware.
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

func gateFor(f *fakeCognito) http.Handler { return composeIdentity(signIn{auth: f.auth()}) }

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

// cookieNamed returns the cookie a response set under name, or nil.
func cookieNamed(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestOnlyHealthAndTheSignInRoutesAreOpen(t *testing.T) {
	h := gateFor(newFakeCognito(t))
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/health", http.StatusOK},
		{http.MethodGet, "/api/auth/login", http.StatusFound},
		{http.MethodGet, "/api/items", http.StatusUnauthorized},
		{http.MethodGet, "/api/me", http.StatusUnauthorized},
		{http.MethodPost, "/api/items", http.StatusUnauthorized},
		// An unknown path is the shell's JSON 404 to everyone, so the smoke check still sees it.
		{http.MethodGet, "/api/nothing-here", http.StatusNotFound},
	} {
		if rec := authDo(t, h, tc.method, tc.path, "", nil); rec.Code != tc.want {
			t.Errorf("%s %s = %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}
}

func TestTheOwnerIsTheVerifiedSub(t *testing.T) {
	f := newFakeCognito(t)
	h := gateFor(f)
	rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(f.idToken("user-a", nil)))
	if rec.Code != http.StatusOK || authDecode(t, rec)["owner"] != "user-a" {
		t.Fatalf("GET /api/items = %d %s, want owner user-a", rec.Code, rec.Body.String())
	}
	// Taken from the cookie too, and /api/me answers the same person.
	rec = authDo(t, h, http.MethodGet, "/api/me", "", withCookies(&http.Cookie{Name: sessionCookie, Value: f.idToken("user-b", nil)}))
	if me := authDecode(t, rec); rec.Code != http.StatusOK || me["sub"] != "user-b" || me["email"] != "user-b@example.invalid" {
		t.Fatalf("GET /api/me = %d %s", rec.Code, rec.Body.String())
	}
}

func TestTheTokenClaimsAreAllChecked(t *testing.T) {
	f := newFakeCognito(t)
	h := gateFor(f)
	other := newTestKey(t)
	in := func(d time.Duration) int64 { return time.Now().Add(d).Unix() }

	for name, token := range map[string]string{
		"another issuer":   f.idToken("intruder", map[string]any{"iss": "https://evil.test/pool"}),
		"another audience": f.idToken("intruder", map[string]any{"aud": "other-client"}),
		"an expired token": f.idToken("intruder", map[string]any{"exp": in(-time.Hour)}),
		"no expiry":        f.idToken("intruder", map[string]any{"exp": nil}),
		"a token_use that is neither id nor access":       f.idToken("intruder", map[string]any{"token_use": "refresh"}),
		"an access token for another client":              f.idToken("intruder", map[string]any{"token_use": "access", "aud": nil, "client_id": "other-client"}),
		"an id token naming the client only in client_id": f.idToken("intruder", map[string]any{"aud": "other-client", "client_id": testClientID}),
		"an iat in the future":                            f.idToken("intruder", map[string]any{"iat": in(time.Hour)}),
		"no subject":                                      f.idToken("", nil),
		"a signature from another key":                    signToken(t, other, testKid, "RS256", f.claims("intruder", nil)),
		"an unsigned token":                               signToken(t, f.key, testKid, "none", f.claims("intruder", nil)),
		"an HS256 token":                                  signToken(t, f.key, testKid, "HS256", f.claims("intruder", nil)),
	} {
		t.Run(name, func(t *testing.T) {
			if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(token)); rec.Code != http.StatusUnauthorized {
				t.Fatalf("GET /api/items = %d, want 401: %s", rec.Code, rec.Body.String())
			}
		})
	}
	// Control: a little clock skew is allowed, so the refusals above are the claims', not the harness's.
	skewed := f.idToken("user-a", map[string]any{"iat": in(time.Minute)})
	if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(skewed)); rec.Code != http.StatusOK {
		t.Fatalf("a token issued a minute ahead = %d, want 200", rec.Code)
	}
}

func newTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestAPoolWhoseKeysCannotBeFetchedIs503NotARefusal(t *testing.T) {
	f := newFakeCognito(t)
	good := f.idToken("user-a", nil)
	a := f.auth()
	a.verifier.jwksURL = f.srv.URL + "/missing-jwks" // the fake answers 400
	h := composeIdentity(signIn{auth: a})
	if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(good)); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("with no keys = %d, want 503", rec.Code)
	}
	// A cookie session is not cleared by it: the person is not signed out by a pool outage.
	rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: good}))
	if rec.Code != http.StatusServiceUnavailable || len(rec.Result().Cookies()) != 0 {
		t.Fatalf("cookie session with no keys = %d, cookies %v; want 503 and none set", rec.Code, rec.Result().Cookies())
	}
}

func TestAKeyRotationDoesNotSignThePersonOut(t *testing.T) {
	f := newFakeCognito(t)
	h := gateFor(f)
	rotated := signToken(t, f.key, "key-2", "RS256", f.claims("user-a", nil))
	rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: rotated}))
	if rec.Code != http.StatusUnauthorized || len(rec.Result().Cookies()) != 0 {
		t.Fatalf("unknown kid = %d, cookies %v; want 401 with the cookies kept", rec.Code, rec.Result().Cookies())
	}
	// A token that is simply forged is signed out.
	forged := signToken(t, newTestKey(t), testKid, "RS256", f.claims("user-a", nil))
	rec = authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: forged}))
	if c := cookieNamed(rec, sessionCookie); rec.Code != http.StatusUnauthorized || c == nil || c.MaxAge >= 0 {
		t.Fatalf("forged token = %d, session cookie %v; want 401 and the cookie cleared", rec.Code, c)
	}
}

func TestAnExpiredSessionIsRefreshedAtTheTokenEndpoint(t *testing.T) {
	f := newFakeCognito(t)
	expired := f.idToken("user-a", map[string]any{"exp": time.Now().Add(-time.Minute).Unix()})
	fresh := f.idToken("user-a", nil)
	f.answerToken(func(form url.Values) (int, any) {
		if form.Get("refresh_token") != "refresh-a" {
			return http.StatusBadRequest, map[string]string{"error": "invalid_grant"}
		}
		return http.StatusOK, map[string]string{"id_token": fresh, "access_token": f.accessToken("user-a", "openid")}
	})
	h := gateFor(f)

	rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(
		&http.Cookie{Name: sessionCookie, Value: expired}, &http.Cookie{Name: refreshCookie, Value: "refresh-a"}))
	if rec.Code != http.StatusOK || authDecode(t, rec)["owner"] != "user-a" {
		t.Fatalf("refresh = %d %s, want 200 for user-a", rec.Code, rec.Body.String())
	}
	if c := cookieNamed(rec, sessionCookie); c == nil || c.Value != fresh || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/" {
		t.Fatalf("the refreshed session cookie = %+v", c)
	}
	call := f.callsTo("/oauth2/token")[0]
	if call.form.Get("grant_type") != "refresh_token" || call.form.Get("client_id") != testClientID || call.header.Get("Authorization") != "" {
		t.Fatalf("refresh request = %v, auth header %q; want the public client's refresh_token grant, no secret", call.form, call.header.Get("Authorization"))
	}

	// A refresh token the endpoint refuses ends the session and clears the cookies.
	rec = authDo(t, h, http.MethodGet, "/api/items", "", withCookies(
		&http.Cookie{Name: sessionCookie, Value: expired}, &http.Cookie{Name: refreshCookie, Value: "revoked"}))
	if c := cookieNamed(rec, sessionCookie); rec.Code != http.StatusUnauthorized || c == nil || c.MaxAge >= 0 {
		t.Fatalf("revoked refresh = %d, session cookie %v; want 401 and cleared", rec.Code, c)
	}
	// An endpoint that is down is a 503 and keeps the session.
	f.answerToken(func(url.Values) (int, any) { return http.StatusBadGateway, map[string]string{} })
	rec = authDo(t, h, http.MethodGet, "/api/items", "", withCookies(
		&http.Cookie{Name: sessionCookie, Value: expired}, &http.Cookie{Name: refreshCookie, Value: "refresh-a"}))
	if rec.Code != http.StatusServiceUnavailable || len(rec.Result().Cookies()) != 0 {
		t.Fatalf("token endpoint down = %d, cookies %v; want 503 and none set", rec.Code, rec.Result().Cookies())
	}
	// A bearer token is never refreshed, and an expired one with no refresh cookie is just refused.
	if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(expired)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired bearer = %d, want 401", rec.Code)
	}
	if rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: expired})); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired cookie with no refresh cookie = %d, want 401", rec.Code)
	}
}

func TestAPublicPostureOpensNothing(t *testing.T) {
	// "public" is accepted from the deploy and treated exactly as "private" in 0.1.0: a deployed
	// function never acts for the anonymous owner.
	env := wholeEnv()
	env[envPosture] = "public"
	env[envLambdaName] = "fn"
	setup := signInFromEnv(getenvFrom(env))
	if setup.err != nil || setup.auth == nil || setup.anonymous {
		t.Fatalf("setup = %+v, want a configured pool and no anonymous mode", setup)
	}
	f := newFakeCognito(t)
	h := composeIdentity(signIn{auth: f.auth()})
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/items"}, {http.MethodHead, "/api/items"}, {http.MethodGet, "/api/me"},
	} {
		if rec := authDo(t, h, tc.method, tc.path, "", nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with no credential = %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
	// The anonymous owner exists only in a local run with no sign-in, never beside a deployed function.
	for name, env := range map[string]map[string]string{
		"a deployed function that is also marked local": {envLambdaName: "fn", envLocalRun: "1"},
		"a deployed function with the local issuer on":  {envLambdaName: "fn", envLocalRun: "1", localauth.EnabledEnv: "1"},
	} {
		if got := signInFromEnv(getenvFrom(env)); got.anonymous || got.err == nil {
			t.Errorf("%s = %+v, want a failure and never anonymous", name, got)
		}
	}
}

func TestEveryTokenThatGetsARequestInIsHeldToTheEmailRule(t *testing.T) {
	f := newFakeCognito(t)
	a := f.auth()
	a.domains = []string{"council.gov.uk"}
	h := composeIdentity(signIn{auth: a})
	good := map[string]any{"email": "sam@council.gov.uk"}

	tokens := map[string]string{
		"a disallowed domain":        f.idToken("user-a", map[string]any{"email": "sam@gmail.com"}),
		"an unverified address":      f.idToken("user-a", map[string]any{"email": "sam@council.gov.uk", "email_verified": false}),
		"no email_verified claim":    f.idToken("user-a", map[string]any{"email": "sam@council.gov.uk", "email_verified": nil}),
		"no email":                   f.idToken("user-a", map[string]any{"email": nil}),
		"a lookalike domain":         f.idToken("user-a", map[string]any{"email": "sam@council.gov.uk.evil.test"}),
		"an access token (no email)": f.accessToken("user-a", "openid email"),
	}
	for name, tok := range tokens {
		t.Run("bearer: "+name, func(t *testing.T) {
			if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(tok)); rec.Code != http.StatusUnauthorized {
				t.Fatalf("= %d %s, want 401", rec.Code, rec.Body.String())
			}
		})
		t.Run("cookie: "+name, func(t *testing.T) {
			rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: tok}))
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("= %d %s, want 401", rec.Code, rec.Body.String())
			}
		})
		t.Run("refresh: "+name, func(t *testing.T) {
			f.answerToken(func(url.Values) (int, any) { return http.StatusOK, map[string]string{"id_token": tok} })
			expired := f.idToken("user-a", map[string]any{"exp": time.Now().Add(-time.Minute).Unix(), "email": "sam@council.gov.uk"})
			rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(
				&http.Cookie{Name: sessionCookie, Value: expired}, &http.Cookie{Name: refreshCookie, Value: "refresh-a"}))
			if c := cookieNamed(rec, sessionCookie); rec.Code != http.StatusUnauthorized || (c != nil && c.MaxAge > 0) {
				t.Fatalf("= %d %s (session cookie %v), want 401 and no session", rec.Code, rec.Body.String(), c)
			}
		})
	}
	// A refused address is told apart from a bad token, and ends a cookie session.
	rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: tokens["a disallowed domain"]}))
	if body := authDecode(t, rec); body["code"] != "email_not_allowed" || body["email"] != nil {
		t.Errorf("body = %v, want code email_not_allowed and no address (no PII)", body)
	}
	if c := cookieNamed(rec, sessionCookie); c == nil || c.MaxAge >= 0 {
		t.Errorf("the refused session cookie was not cleared: %v", c)
	}

	// Controls: an allowed, verified ID token passes by bearer and cookie, mixed case and padding included.
	ok := f.idToken("user-a", map[string]any{"email": " Sam@Council.GOV.uk "})
	if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(ok)); rec.Code != http.StatusOK {
		t.Errorf("allowed bearer = %d", rec.Code)
	}
	if rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: f.idToken("user-a", good)})); rec.Code != http.StatusOK {
		t.Errorf("allowed cookie = %d", rec.Code)
	}
	// With no rule set, a verified ID token passes and an unverified one still does not.
	open := composeIdentity(signIn{auth: f.auth()})
	if rec := authDo(t, open, http.MethodGet, "/api/items", "", withBearer(tokens["an unverified address"])); rec.Code != http.StatusUnauthorized {
		t.Errorf("unverified with no domain rule = %d, want 401", rec.Code)
	}
}

// wholeEnv is a complete, valid environment for a hosted pool.
func wholeEnv() map[string]string {
	return map[string]string{
		envIssuer:    "https://cognito-idp.eu-west-2.amazonaws.com/eu-west-2_abc123",
		envClientID:  testClientID,
		envJWKSURL:   "https://cognito-idp.eu-west-2.amazonaws.com/eu-west-2_abc123/.well-known/jwks.json",
		envPosture:   "private",
		envLoginURL:  "https://pool.auth.eu-west-2.amazoncognito.com/oauth2/authorize?client_id=" + testClientID + "&response_type=code&scope=openid%20email&redirect_uri=https%3A%2F%2Fapp.test%2Fapi%2Fauth%2Fcallback",
		envLogoutURL: testLogoutURL,
	}
}

func getenvFrom(env map[string]string) func(string) string {
	return func(k string) string { return env[k] }
}

func TestAMissingOrBadVariableClosesEveryAPIRoute(t *testing.T) {
	// The control: the whole environment is accepted, and the config reads as the module wrote it.
	setup := signInFromEnv(getenvFrom(wholeEnv()))
	if setup.err != nil || setup.auth == nil {
		t.Fatalf("a complete environment = %+v", setup)
	}
	if setup.auth.tokenURL != "https://pool.auth.eu-west-2.amazoncognito.com/oauth2/token" ||
		setup.auth.revokeURL != "https://pool.auth.eu-west-2.amazoncognito.com/oauth2/revoke" ||
		setup.auth.redirectURI != testRedirectURI ||
		setup.auth.endpoint != "https://cognito-idp.eu-west-2.amazonaws.com/" {
		t.Fatalf("endpoints derived wrongly: %+v", setup.auth)
	}

	cases := map[string]func(map[string]string){
		"no AUTH_ISSUER":                 func(e map[string]string) { delete(e, envIssuer) },
		"no AUTH_CLIENT_ID":              func(e map[string]string) { delete(e, envClientID) },
		"no AUTH_JWKS_URL":               func(e map[string]string) { delete(e, envJWKSURL) },
		"no AUTH_POSTURE":                func(e map[string]string) { delete(e, envPosture) },
		"no AUTH_LOGIN_URL":              func(e map[string]string) { delete(e, envLoginURL) },
		"no AUTH_LOGOUT_URL":             func(e map[string]string) { delete(e, envLogoutURL) },
		"an unknown posture":             func(e map[string]string) { e[envPosture] = "open" },
		"an http issuer":                 func(e map[string]string) { e[envIssuer] = "http://cognito-idp.eu-west-2.amazonaws.com/p" },
		"an http JWKS URL":               func(e map[string]string) { e[envJWKSURL] = "http://pool.test/jwks.json" },
		"a login URL for another client": func(e map[string]string) { e[envLoginURL] = strings.Replace(e[envLoginURL], testClientID, "other", 1) },
		"a login URL that already has a state": func(e map[string]string) {
			e[envLoginURL] += "&state=fixed"
		},
		"a callback that is not the API's": func(e map[string]string) {
			e[envLoginURL] = strings.Replace(e[envLoginURL], "%2Fapi%2Fauth%2Fcallback", "%2Felsewhere", 1)
		},
		"an http callback": func(e map[string]string) {
			e[envLoginURL] = strings.Replace(e[envLoginURL], "https%3A", "http%3A", 1)
		},
		"a logout URL for another client": func(e map[string]string) {
			e[envLogoutURL] = strings.Replace(e[envLogoutURL], testClientID, "other", 1)
		},
		"a malformed SIGNUP_DOMAINS": func(e map[string]string) { e[envSignupDomains] = "council.gov.uk, not a domain!" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			env := wholeEnv()
			mutate(env)
			setup := signInFromEnv(getenvFrom(env))
			if setup.err == nil || setup.auth != nil || setup.anonymous {
				t.Fatalf("setup = %+v, want an error and no auth", setup)
			}
			h := composeIdentity(setup)
			for _, path := range []string{"/api/items", "/api/me", "/api/auth/login", "/api/auth/callback"} {
				if rec := authDo(t, h, http.MethodGet, path, "", nil); rec.Code != http.StatusServiceUnavailable {
					t.Errorf("GET %s = %d, want 503", path, rec.Code)
				}
			}
			if rec := authDo(t, h, http.MethodPost, "/api/auth/sign-out", "{}", nil); rec.Code != http.StatusServiceUnavailable {
				t.Errorf("POST /api/auth/sign-out = %d, want 503", rec.Code)
			}
			// Only the health check stays up, so the deploy can see the function is alive.
			if rec := authDo(t, h, http.MethodGet, "/api/health", "", nil); rec.Code != http.StatusOK {
				t.Errorf("GET /api/health = %d, want 200", rec.Code)
			}
		})
	}
}

func TestWhereIdentityComesFrom(t *testing.T) {
	for name, tc := range map[string]struct {
		env           map[string]string
		wantErr       bool
		wantAnonymous bool
		wantLocal     bool
	}{
		"a deployed function with no pool is a failure": {env: map[string]string{envLambdaName: "fn", envLocalRun: "1", localauth.EnabledEnv: "1"}, wantErr: true},
		"neither local nor configured is a failure":     {env: map[string]string{}, wantErr: true},
		"a local run is anonymous":                      {env: map[string]string{envLocalRun: "1"}, wantAnonymous: true},
		"a local run with the issuer trusts it":         {env: map[string]string{envLocalRun: "1", localauth.EnabledEnv: "1"}, wantLocal: true},
		"a named pool never trusts the local issuer": {
			env: func() map[string]string {
				e := wholeEnv()
				e[envLocalRun], e[localauth.EnabledEnv] = "1", "1"
				return e
			}(),
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := signInFromEnv(getenvFrom(tc.env))
			if (got.err != nil) != tc.wantErr || got.anonymous != tc.wantAnonymous || (got.auth != nil && got.auth.local) != tc.wantLocal {
				t.Fatalf("signInFromEnv = %+v, want err=%v anonymous=%v local=%v", got, tc.wantErr, tc.wantAnonymous, tc.wantLocal)
			}
		})
	}
}

func TestALocalRunHasNoHostedSignIn(t *testing.T) {
	h := composeIdentity(signIn{auth: newLocalIssuerAuth("http://localhost:8080/api/_local/auth")})
	for _, path := range []string{"/api/auth/login", "/api/auth/callback"} {
		rec := authDo(t, h, http.MethodGet, path, "", nil)
		if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/sign-in?error=local" {
			t.Errorf("GET %s = %d %q, want a redirect to the page that says why", path, rec.Code, rec.Header().Get("Location"))
		}
	}
	if rec := authDo(t, h, http.MethodPost, "/api/auth/sign-out", "{}", nil); rec.Code != http.StatusNotImplemented {
		t.Errorf("POST sign-out = %d, want 501", rec.Code)
	}
	// And with no sign-in at all, every request acts for the anonymous owner.
	anon := composeIdentity(signIn{anonymous: true})
	if rec := authDo(t, anon, http.MethodGet, "/api/items", "", nil); rec.Code != http.StatusOK || authDecode(t, rec)["owner"] != identity.AnonymousOwner {
		t.Errorf("anonymous run = %d %s", rec.Code, rec.Body.String())
	}
	if rec := authDo(t, anon, http.MethodGet, "/api/auth/login", "", nil); rec.Code != http.StatusFound {
		t.Errorf("anonymous run GET login = %d, want the local-run redirect", rec.Code)
	}
}
