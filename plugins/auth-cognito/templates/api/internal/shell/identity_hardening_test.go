package shell

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// These tests pin the checks the identity seam makes that a green happy path would not notice
// losing: each one fails when the check it is named for is removed.

const stubIssuer = "https://pool.test/pool"

// jwksStub serves a JWKS from handler and counts the fetches.
type jwksStub struct {
	srv  *httptest.Server
	hits atomic.Int32
}

func newJWKSStub(t *testing.T, handler func(w http.ResponseWriter, hit int)) *jwksStub {
	t.Helper()
	s := &jwksStub{}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handler(w, int(s.hits.Add(1)))
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *jwksStub) verifier() *cognitoVerifier {
	return newCognitoVerifier(stubIssuer, testClientID, s.srv.URL, s.srv.Client())
}

// jwksJSON is a JWKS holding key under kid, with the field values overridable.
func jwksJSON(key *rsa.PrivateKey, kid string, override map[string]any) []byte {
	pub := key.PublicKey
	k := map[string]any{
		"kid": kid, "kty": "RSA", "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
	for name, v := range override {
		k[name] = v
	}
	b, _ := json.Marshal(map[string]any{"keys": []any{k}})
	return b
}

func stubClaims(override map[string]any) map[string]any {
	now := time.Now()
	c := map[string]any{
		"iss": stubIssuer, "sub": "user-a", "aud": testClientID, "token_use": "id",
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
	}
	for k, v := range override {
		c[k] = v
	}
	return c
}

func newTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestTheTokenClaimsAreAllChecked(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	in := func(d time.Duration) int64 { return time.Now().Add(d).Unix() }

	for name, token := range map[string]string{
		"an access token for another client":              f.token("intruder", map[string]any{"token_use": "access", "aud": nil, "client_id": "other-client"}),
		"an access token with no client":                  f.token("intruder", map[string]any{"token_use": "access", "aud": nil}),
		"an id token naming the client only in client_id": f.token("intruder", map[string]any{"aud": "other-client", "client_id": testClientID}),
		"an iat in the future":                            f.token("intruder", map[string]any{"iat": in(time.Hour)}),
		"no expiry":                                       f.token("intruder", map[string]any{"exp": 0}),
		"no subject":                                      f.token("", nil),
	} {
		t.Run(name, func(t *testing.T) {
			if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(token)); rec.Code != http.StatusUnauthorized {
				t.Fatalf("GET /api/items = %d, want 401: %s", rec.Code, rec.Body.String())
			}
		})
	}
	// Control: a little clock skew is allowed, so the refusals above are the claims', not the harness's.
	skewed := f.token("user-a", map[string]any{"iat": in(time.Minute)})
	if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(skewed)); rec.Code != http.StatusOK {
		t.Fatalf("a token issued a minute ahead = %d, want 200", rec.Code)
	}
}

func TestTheKeysAreNotRefetchedForEveryUnknownKid(t *testing.T) {
	key := newTestKey(t)
	stub := newJWKSStub(t, func(w http.ResponseWriter, _ int) { _, _ = w.Write(jwksJSON(key, "k1", nil)) })
	v := stub.verifier()
	clock := time.Now()
	v.now = func() time.Time { return clock }

	token := func(kid string) string { return signToken(t, key, kid, "RS256", stubClaims(nil)) }
	if _, err := v.verify(context.Background(), token("k1")); err != nil {
		t.Fatal(err)
	}
	for _, kid := range []string{"forged-1", "forged-2", "forged-3"} {
		clock = clock.Add(time.Second)
		if _, err := v.verify(context.Background(), token(kid)); !errors.Is(err, errUnknownKey) {
			t.Fatalf("kid %s = %v, want an unknown key", kid, err)
		}
	}
	if got := stub.hits.Load(); got != 1 {
		t.Fatalf("the JWKS was fetched %d times for forged kids inside the interval, want 1", got)
	}
	clock = clock.Add(jwksRefetchInterval + time.Second)
	_, _ = v.verify(context.Background(), token("forged-4"))
	if got := stub.hits.Load(); got != 2 {
		t.Fatalf("the JWKS was fetched %d times after the interval, want 2", got)
	}
}

func TestAPoolThatCannotBeReachedIs503NotARefusal(t *testing.T) {
	key := newTestKey(t)
	var down atomic.Bool
	down.Store(true)
	stub := newJWKSStub(t, func(w http.ResponseWriter, _ int) {
		if down.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(jwksJSON(key, "k1", nil))
	})
	v := stub.verifier()
	clock := time.Now()
	v.now = func() time.Time { return clock }
	h := composeIdentity(signIn{auth: &cognitoAuth{clientID: testClientID, client: stub.srv.Client(), verifier: v}})
	good := signToken(t, key, "k1", "RS256", stubClaims(nil))

	for i := 0; i < 3; i++ {
		rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(good))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("a valid token while the JWKS is down = %d, want 503 (the person is not at fault)", rec.Code)
		}
	}
	if got := stub.hits.Load(); got != 1 {
		t.Fatalf("a JWKS that is down was fetched %d times for 3 requests, want 1", got)
	}

	// A cookie session survives the outage: it is neither cleared nor read as signed out.
	rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: good}))
	if rec.Code != http.StatusServiceUnavailable || len(rec.Result().Cookies()) != 0 {
		t.Fatalf("a cookie session during an outage = %d with cookies %v", rec.Code, rec.Result().Cookies())
	}

	down.Store(false)
	clock = clock.Add(jwksFailureTTL + time.Second)
	if rec := authDo(t, h, http.MethodGet, "/api/items", "", withBearer(good)); rec.Code != http.StatusOK {
		t.Fatalf("after the pool recovers = %d, want 200", rec.Code)
	}
}

func TestAJWKSIsOnlyTrustedWhenItIsWellFormed(t *testing.T) {
	key := newTestKey(t)
	pub := key.PublicKey
	fiveByteExponent := base64.RawURLEncoding.EncodeToString(append([]byte{0, 0}, big.NewInt(int64(pub.E)).Bytes()...))
	padded := append([]byte(`{"pad":"`), []byte(strings.Repeat("a", maxJWKSBytes+1024))...)
	padded = append(padded, []byte(`","keys":`)...)
	padded = append(padded, jwksJSON(key, "k1", nil)[len(`{"keys":`):]...)

	cases := map[string]struct {
		status int
		body   []byte
		ok     bool
	}{
		"a well-formed JWKS (control)":    {http.StatusOK, jwksJSON(key, "k1", nil), true},
		"an error status with valid keys": {http.StatusInternalServerError, jwksJSON(key, "k1", nil), false},
		"a key for another algorithm":     {http.StatusOK, jwksJSON(key, "k1", map[string]any{"alg": "HS256"}), false},
		"a key of another type":           {http.StatusOK, jwksJSON(key, "k1", map[string]any{"kty": "oct"}), false},
		"an exponent longer than 4 bytes": {http.StatusOK, jwksJSON(key, "k1", map[string]any{"e": fiveByteExponent}), false},
		"a body over the size cap":        {http.StatusOK, padded, false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			stub := newJWKSStub(t, func(w http.ResponseWriter, _ int) {
				w.WriteHeader(c.status)
				_, _ = w.Write(c.body)
			})
			_, err := stub.verifier().verify(context.Background(), signToken(t, key, "k1", "RS256", stubClaims(nil)))
			switch {
			case c.ok && err != nil:
				t.Fatalf("verify = %v, want the token accepted", err)
			case !c.ok && !errors.Is(err, errVerifierUnavailable):
				t.Fatalf("verify = %v, want the keys unavailable", err)
			}
		})
	}
}

func TestAKeyRotationDoesNotSignThePersonOut(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	// The first fetch after a rotation finds the new kid missing too (the pool lags its own tokens).
	early := &http.Cookie{Name: sessionCookie, Value: signToken(t, f.key, "key-2", "RS256", f.claims("user-a", nil))}
	if rec := authDo(t, gateFor(f, srv), http.MethodGet, "/api/items", "", withCookies(early)); rec.Code != http.StatusUnauthorized || len(rec.Result().Cookies()) != 0 {
		t.Fatalf("an unknown key on a fresh fetch = %d with cookies %v, want 401 and none", rec.Code, rec.Result().Cookies())
	}
	session := &http.Cookie{Name: sessionCookie, Value: f.token("user-a", nil)}
	if rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(session)); rec.Code != http.StatusOK {
		t.Fatalf("GET /api/items = %d", rec.Code)
	}

	// A token signed under a kid the pool has not published yet (a rotation): refused, not ended.
	rotated := &http.Cookie{Name: sessionCookie, Value: signToken(t, f.key, "key-2", "RS256", f.claims("user-a", nil))}
	rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(rotated, &http.Cookie{Name: refreshCookie, Value: "refresh-a"}))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("a rotated key = %d, want 401", rec.Code)
	}
	if cookies := rec.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("an unknown key cleared the session: %v", cookies)
	}

	// A token that is wrong, by contrast, does end the session.
	other := newTestKey(t)
	forged := &http.Cookie{Name: sessionCookie, Value: signToken(t, other, testKid, "RS256", f.claims("user-a", nil))}
	rec = authDo(t, h, http.MethodGet, "/api/items", "", withCookies(forged))
	if c := responseCookie(rec, sessionCookie); rec.Code != http.StatusUnauthorized || c == nil || c.MaxAge >= 0 {
		t.Fatalf("a forged cookie = %d, cookies %v, want 401 and the session cleared", rec.Code, rec.Result().Cookies())
	}
}

func TestOnlyAnExpiredCookieSessionIsRefreshed(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	var refreshes atomic.Int32
	f.answer("InitiateAuth", func(map[string]any) (int, any) {
		refreshes.Add(1)
		return http.StatusOK, map[string]any{"AuthenticationResult": map[string]string{"IdToken": f.token("user-a", nil)}}
	})
	expired := f.token("user-a", map[string]any{"exp": time.Now().Add(-time.Minute).Unix()})
	refresh := &http.Cookie{Name: refreshCookie, Value: "refresh-a"}
	other := newTestKey(t)

	cases := map[string]func(*http.Request){
		"an expired bearer token":       func(r *http.Request) { withBearer(expired)(r); withCookies(refresh)(r) },
		"an invalid cookie token":       withCookies(&http.Cookie{Name: sessionCookie, Value: signToken(t, other, testKid, "RS256", f.claims("user-a", nil))}, refresh),
		"a token for another client":    withCookies(&http.Cookie{Name: sessionCookie, Value: f.token("user-a", map[string]any{"aud": "other-client", "exp": time.Now().Add(-time.Minute).Unix()})}, refresh),
		"a refresh cookie and no token": withCookies(refresh),
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			rec := authDo(t, h, http.MethodGet, "/api/items", "", edit)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("GET /api/items = %d, want 401", rec.Code)
			}
		})
	}
	if got := refreshes.Load(); got != 0 {
		t.Fatalf("the refresh token was used %d times for tokens that were not genuinely expired", got)
	}

	rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: expired}, refresh))
	if rec.Code != http.StatusOK || refreshes.Load() != 1 {
		t.Fatalf("a genuinely expired cookie session = %d with %d refreshes, want 200 and 1", rec.Code, refreshes.Load())
	}
	if c := responseCookie(rec, sessionCookie); c == nil || c.Path != "/" || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode {
		t.Fatalf("the refreshed session cookie = %+v", c)
	}
}

func TestTheRefreshCookieIsKeptToTheAPI(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	f.answer("InitiateAuth", func(map[string]any) (int, any) {
		return http.StatusOK, map[string]any{"AuthenticationResult": map[string]string{"IdToken": f.token("user-a", nil), "RefreshToken": "refresh-a"}}
	})
	rec := authDo(t, h, http.MethodPost, "/api/auth/sign-in", `{"email":"a@example.invalid","password":"pw"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("sign-in = %d: %s", rec.Code, rec.Body.String())
	}
	if c := responseCookie(rec, refreshCookie); c == nil || c.Path != "/api" {
		t.Fatalf("refresh cookie = %+v, want Path=/api so no page request carries it", c)
	}
	if c := responseCookie(rec, sessionCookie); c == nil || c.Path != "/" {
		t.Fatalf("session cookie = %+v, want Path=/", c)
	}
	out := authDo(t, h, http.MethodPost, "/api/auth/sign-out", "", nil)
	if c := responseCookie(out, refreshCookie); c == nil || c.Path != "/api" || c.MaxAge >= 0 {
		t.Fatalf("sign-out must clear the refresh cookie at its own path: %+v", c)
	}
}

func TestANonBearerAuthorizationHeaderIsNotMixedWithACookie(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	valid := &http.Cookie{Name: sessionCookie, Value: f.token("user-a", nil)}
	for name, header := range map[string]string{
		"basic":                  "Basic dXNlcjpwYXNz",
		"a lower-case scheme":    "bearer " + f.token("user-a", nil),
		"a scheme with no token": "Digest",
	} {
		t.Run(name, func(t *testing.T) {
			rec := authDo(t, h, http.MethodGet, "/api/items", "", func(r *http.Request) {
				r.Header.Set("Authorization", header)
				r.AddCookie(valid)
			})
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("GET /api/items = %d, want 401: a header that is not a bearer token must not fall back to the cookie", rec.Code)
			}
		})
	}
}

func TestTheAuthBodyIsCapped(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	f.answer("InitiateAuth", func(map[string]any) (int, any) { return http.StatusOK, map[string]any{} })

	big := `{"email":"a@example.invalid","password":"` + strings.Repeat("x", 2*maxAuthBodyBytes) + `"}`
	rec := authDo(t, h, http.MethodPost, "/api/auth/sign-in", big, nil)
	if rec.Code != http.StatusBadRequest || authDecode(t, rec)["code"] != "invalid_input" {
		t.Fatalf("an oversized sign-in body = %d %s, want 400 invalid_input", rec.Code, rec.Body.String())
	}
	if f.called("InitiateAuth") {
		t.Fatal("an oversized body reached the pool")
	}
}
