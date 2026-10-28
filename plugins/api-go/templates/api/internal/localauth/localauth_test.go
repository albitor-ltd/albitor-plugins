package localauth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func envOf(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func localEnv() map[string]string {
	return map[string]string{LocalModeEnv: "1", EnabledEnv: "1", "PORT": "8080"}
}

func mustOpen(t *testing.T) *Issuer {
	t.Helper()
	l, err := Open(envOf(localEnv()))
	if err != nil || l == nil {
		t.Fatalf("Open = %v, %v; want an issuer", l, err)
	}
	return l
}

func TestOpenStartsTheIssuerInALocalRun(t *testing.T) {
	l := mustOpen(t)
	if got, want := l.URL(), "http://localhost:8080"+Path; got != want {
		t.Fatalf("issuer = %q, want %q", got, want)
	}
}

func TestOpenStartsNothingUnlessAskedWithOne(t *testing.T) {
	for _, v := range []string{"", "0", "true", "yes"} {
		l, err := Open(envOf(map[string]string{LocalModeEnv: "1", EnabledEnv: v}))
		if err != nil || l != nil {
			t.Fatalf("%s=%q: Open = %v, %v; want nothing started", EnabledEnv, v, l, err)
		}
	}
}

// Each guard refuses on its own, and a refusal starts nothing.
func TestOpenRefusesTheIssuerOutsideALocalRun(t *testing.T) {
	with := func(extra map[string]string) map[string]string {
		vars := localEnv()
		for k, v := range extra {
			vars[k] = v
		}
		return vars
	}
	cases := map[string]struct {
		vars map[string]string
		want error
	}{
		"without local mode":           {map[string]string{EnabledEnv: "1"}, ErrNotLocal},
		"local mode not 1":             {map[string]string{EnabledEnv: "1", LocalModeEnv: "true"}, ErrNotLocal},
		"on a deployed function":       {with(map[string]string{"AWS_LAMBDA_FUNCTION_NAME": "app-api"}), ErrDeployed},
		"beside an issuer":             {with(map[string]string{"AUTH_ISSUER": "https://cognito-idp.eu-west-2.amazonaws.com/p"}), ErrRealPool},
		"beside a pool id":             {with(map[string]string{"AUTH_USER_POOL_ID": "eu-west-2_x"}), ErrRealPool},
		"beside a client id":           {with(map[string]string{"AUTH_CLIENT_ID": "c"}), ErrRealPool},
		"deployed, with a pool, no lm": {map[string]string{EnabledEnv: "1", "AUTH_USER_POOL_ID": "p", "AWS_LAMBDA_FUNCTION_NAME": "f"}, ErrRealPool},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			l, err := Open(envOf(tc.vars))
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if l != nil {
				t.Fatal("an issuer was started alongside the refusal")
			}
		})
	}
}

// keysFrom reads the issuer's JWKS as a verifier would.
func keysFrom(t *testing.T, doc []byte) map[string]*rsa.PublicKey {
	t.Helper()
	var set struct {
		Keys []struct{ Kty, Kid, Use, Alg, N, E string } `json:"keys"`
	}
	if err := json.Unmarshal(doc, &set); err != nil {
		t.Fatalf("JWKS: %v", err)
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" || k.Use != "sig" || k.Alg != "RS256" {
			t.Fatalf("key %+v is not an RS256 signing key", k)
		}
		n, _ := base64.RawURLEncoding.DecodeString(k.N)
		e, _ := base64.RawURLEncoding.DecodeString(k.E)
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	return keys
}

// claimsOf checks a token's RS256 signature against keys and returns its
// header and claims.
func claimsOf(t *testing.T, keys map[string]*rsa.PublicKey, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts", len(parts))
	}
	var header struct{ Alg, Kid string }
	raw, _ := base64.RawURLEncoding.DecodeString(parts[0])
	_ = json.Unmarshal(raw, &header)
	pub, ok := keys[header.Kid]
	if header.Alg != "RS256" || !ok {
		t.Fatalf("header %+v names no published RS256 key", header)
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
		t.Fatalf("signature does not verify against the JWKS: %v", err)
	}
	var claims map[string]any
	raw, _ = base64.RawURLEncoding.DecodeString(parts[1])
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatal(err)
	}
	return claims
}

// Over HTTP, as a spec uses it: two users get tokens for two subjects, shaped
// as Cognito's and signed by the key the JWKS publishes.
func TestTokensAreCognitoShapedAndVerifyAgainstTheJWKS(t *testing.T) {
	l := mustOpen(t)
	app := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	srv := httptest.NewServer(l.Wrap(app))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + Path + "/.well-known/jwks.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	keys := keysFrom(t, doc)

	mint := func(name string) Tokens {
		resp, err := http.Post(srv.URL+Path+"/token", "application/json", strings.NewReader(`{"identity":"`+name+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("token for %s: status %d", name, resp.StatusCode)
		}
		var tokens Tokens
		if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
			t.Fatal(err)
		}
		return tokens
	}
	alice, bob := mint("alice"), mint("bob")

	aliceID := claimsOf(t, keys, alice.IDToken)
	bobID := claimsOf(t, keys, bob.IDToken)
	if aliceID["sub"] == "" || aliceID["sub"] == bobID["sub"] {
		t.Fatalf("subjects alice=%v bob=%v, want two different ones", aliceID["sub"], bobID["sub"])
	}
	for k, want := range map[string]any{"iss": l.URL(), "aud": ClientID, "token_use": "id", "email": "alice@example.test"} {
		if aliceID[k] != want {
			t.Errorf("ID token %s = %v, want %v", k, aliceID[k], want)
		}
	}
	access := claimsOf(t, keys, alice.AccessToken)
	for k, want := range map[string]any{"iss": l.URL(), "client_id": ClientID, "token_use": "access", "sub": aliceID["sub"]} {
		if access[k] != want {
			t.Errorf("access token %s = %v, want %v", k, access[k], want)
		}
	}
	if _, ok := aliceID["exp"].(float64); !ok {
		t.Error("ID token has no exp")
	}
	adminClaims := claimsOf(t, keys, mint("admin").IDToken)
	if groups, _ := adminClaims["cognito:groups"].([]any); len(groups) != 1 || groups[0] != "admin" {
		t.Errorf("admin groups = %v, want [admin]", adminClaims["cognito:groups"])
	}
}

func TestWrapLeavesTheAppAloneAndOwnsOnlyItsPath(t *testing.T) {
	l := mustOpen(t)
	h := l.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }))
	for target, want := range map[string]int{
		"/api/items":                    http.StatusTeapot,
		"/api/_local/authx":             http.StatusTeapot,
		Path + "/identities":            http.StatusOK,
		Path + "/../../items":           http.StatusNotFound,
		Path + "/nothing":               http.StatusNotFound,
		Path + "/.well-known/jwks.json": http.StatusOK,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != want {
			t.Errorf("GET %s = %d, want %d", target, rec.Code, want)
		}
	}
	for body, want := range map[string]int{`{"identity":"mallory"}`: http.StatusNotFound, `{`: http.StatusBadRequest} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, Path+"/token", strings.NewReader(body)))
		if rec.Code != want {
			t.Errorf("POST token %s = %d, want %d", body, rec.Code, want)
		}
	}
}

// Two issuers never share a key: the key is generated per process.
func TestKeysArePerIssuer(t *testing.T) {
	first, second := mustOpen(t), mustOpen(t)
	if first.kid == second.kid || first.key.N.Cmp(second.key.N) == 0 {
		t.Fatal("two issuers share a signing key")
	}
}

func TestSeedRoleRegistersOneIdentityHoldingOnlyItsOwnGroup(t *testing.T) {
	l := mustOpen(t)
	if _, err := l.Mint(RoleIdentityName("manager")); err == nil {
		t.Fatal("a role is signable before it is seeded")
	}
	first, err := l.SeedRole("manager")
	if err != nil {
		t.Fatal(err)
	}
	again, _ := l.SeedRole("manager")
	other, _ := l.SeedRole("reviewer")
	if first.Subject != again.Subject || first.Subject == other.Subject {
		t.Fatalf("seeding is not idempotent or subjects collide: %+v %+v %+v", first, again, other)
	}
	if len(first.Groups) != 1 || first.Groups[0] != "manager" {
		t.Fatalf("groups = %v, want only [manager]", first.Groups)
	}
	tokens, err := l.Mint(first.Name)
	if err != nil {
		t.Fatal(err)
	}
	claims := claimsOf(t, keysFrom(t, l.jwks), tokens.IDToken)
	if g, _ := claims["cognito:groups"].([]any); len(g) != 1 || g[0] != "manager" {
		t.Errorf("token groups = %v, want [manager]", claims["cognito:groups"])
	}
	if claims["sub"] != first.Subject {
		t.Errorf("sub = %v, want %s", claims["sub"], first.Subject)
	}
	// A fresh issuer derives the same subject: a record written last run is the same user's now.
	again2, _ := mustOpen(t).SeedRole("manager")
	if again2.Subject != first.Subject {
		t.Errorf("subject differs across issuers: %s vs %s", again2.Subject, first.Subject)
	}
}

func TestSeedRoleRefusesWhatIsNotAGroupName(t *testing.T) {
	l := mustOpen(t)
	for _, bad := range []string{"", "A", "x", "1abc", "has space", "under_score", strings.Repeat("a", 33)} {
		if _, err := l.SeedRole(bad); !errors.Is(err, ErrBadRole) {
			t.Errorf("SeedRole(%q) = %v, want ErrBadRole", bad, err)
		}
	}
}

func TestSeedRouteSeedsAndListsRoleIdentities(t *testing.T) {
	l := mustOpen(t)
	post := func(path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		l.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		return rec
	}
	if rec := post(Path+"/seed", `{"role":"manager"}`); rec.Code != http.StatusOK {
		t.Fatalf("seed = %d %s", rec.Code, rec.Body)
	}
	if rec := post(Path+"/seed", `{"role":"Bad Role"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a bad role = %d, want 400", rec.Code)
	}
	if rec := post(Path+"/token", `{"identity":"role-manager"}`); rec.Code != http.StatusOK {
		t.Errorf("token for a seeded role = %d", rec.Code)
	}
	rec := httptest.NewRecorder()
	l.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Path+"/identities", nil))
	if !strings.Contains(rec.Body.String(), `"role-manager"`) || !strings.Contains(rec.Body.String(), `"alice"`) {
		t.Errorf("identities = %s", rec.Body)
	}
}
