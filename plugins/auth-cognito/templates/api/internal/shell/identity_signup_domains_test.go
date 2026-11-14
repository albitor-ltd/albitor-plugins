package shell

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseSignupDomains(t *testing.T) {
	got, err := parseSignupDomains("Example.org, *.dept.example.org\n@other.co.uk")
	if err != nil || strings.Join(got, "|") != "example.org|*.dept.example.org|other.co.uk" {
		t.Fatalf("parse = %v, %v", got, err)
	}
	if got, err := parseSignupDomains("  "); err != nil || got != nil {
		t.Fatalf("empty = %v, %v; want no rule", got, err)
	}
	for _, bad := range []string{"example", "*example.org", "a@b.org", "ex ample..org", "*.org."} {
		if _, err := parseSignupDomains(bad); err == nil {
			t.Errorf("%q should be refused so a typo closes the API", bad)
		}
	}
}

func TestSignInFromEnvRefusesAMalformedSignupDomain(t *testing.T) {
	env := map[string]string{
		"AUTH_USER_POOL_ID": "eu-west-2_abc", "AUTH_CLIENT_ID": "client", "AWS_REGION": "eu-west-2",
		"AUTH_SIGNUP_DOMAINS": "not a domain",
	}
	if s := signInFromEnv(identityTestEnv(env)); s.err == nil || s.auth != nil {
		t.Fatalf("a malformed AUTH_SIGNUP_DOMAINS = %+v, want the API closed", s)
	}
	env["AUTH_SIGNUP_DOMAINS"] = "example.org"
	if s := signInFromEnv(identityTestEnv(env)); s.err != nil || s.auth == nil || len(s.auth.signupDomains) != 1 {
		t.Fatalf("a valid AUTH_SIGNUP_DOMAINS = %+v, want it kept", s)
	}
}

func TestSignupAllowed(t *testing.T) {
	cases := []struct {
		domains []string
		email   string
		want    bool
	}{
		{nil, "anyone@anywhere.test", true},
		{[]string{"example.org"}, "a@example.org", true},
		{[]string{"example.org"}, "a@EXAMPLE.org", true},
		{[]string{"example.org"}, "a@sub.example.org", false},
		{[]string{"example.org"}, "a@notexample.org", false},
		{[]string{"*.example.org"}, "a@sub.example.org", true},
		{[]string{"*.example.org"}, "a@example.org", false},
		{[]string{"*.example.org"}, "a@evilexample.org", false},
		{[]string{"example.org", "other.org"}, "a@other.org", true},
		{[]string{"example.org"}, "no-at-sign", false},
		{[]string{"example.org"}, "@example.org", false},
		{[]string{"example.org"}, "a@example.org.evil.com", false},
		{[]string{"*.example.org"}, "a@x.example.org.evil.com", false},
		{[]string{"example.org"}, "a@example.org.", false},
		{[]string{"*.example.org"}, "a@x.example.org.", false},
		{[]string{"xn--bcher-kva.example"}, "a@xn--bcher-kva.example", true},
	}
	for _, c := range cases {
		a := &cognitoAuth{signupDomains: c.domains}
		if got := a.signupAllowed(c.email); got != c.want {
			t.Errorf("signupAllowed(%v, %q) = %v, want %v", c.domains, c.email, got, c.want)
		}
	}
}

// The list holds every token, not only the sign-up call: a person who registered straight with the
// pool's public client id (never touching /api/auth/sign-up) is refused by the API, as is an
// invitee or an existing account on another domain, an access token (no address) and a token whose
// address was moved. Listed addresses and the platform-seeded group pass, and with no list nothing changes.
func TestTheDomainListHoldsEveryToken(t *testing.T) {
	f, srv := newFakeCognito(t)
	gate := func(domains []string) http.Handler {
		return composeIdentity(signIn{auth: &cognitoAuth{
			clientID: testClientID, endpoint: srv.URL + "/", client: srv.Client(), signupDomains: domains,
			verifier: newCognitoVerifier(f.issuer, testClientID, f.issuer+"/.well-known/jwks.json", srv.Client()),
		}})
	}
	limited := gate([]string{"example.org"})
	get := func(h http.Handler, token string, cookie bool) *httptest.ResponseRecorder {
		if cookie {
			return authDo(t, h, http.MethodGet, "/api/me", "", withCookies(&http.Cookie{Name: sessionCookie, Value: token}))
		}
		return authDo(t, h, http.MethodGet, "/api/me", "", withBearer(token))
	}
	with := func(email string) string { return f.token("u", map[string]any{"email": email}) }

	for _, email := range []string{"direct@other.test", "invitee@other.test", "x@example.org.evil.com", ""} {
		rec := get(limited, with(email), false)
		if rec.Code != http.StatusForbidden || authDecode(t, rec)["code"] != "email_not_allowed" {
			t.Fatalf("%q = %d %s, want 403 email_not_allowed", email, rec.Code, rec.Body.String())
		}
	}
	access := f.token("u", map[string]any{"token_use": "access", "client_id": testClientID})
	if rec := get(limited, access, false); rec.Code != http.StatusForbidden {
		t.Fatalf("access token under a list = %d, want 403", rec.Code)
	}
	rec := get(limited, with("other@other.test"), true)
	if rec.Code != http.StatusForbidden || responseCookie(rec, sessionCookie) == nil || responseCookie(rec, sessionCookie).MaxAge >= 0 {
		t.Fatalf("a refused session cookie = %d %v, want 403 and the cookie cleared", rec.Code, rec.Result().Cookies())
	}
	if rec := get(limited, with("A@Example.ORG"), false); rec.Code != http.StatusOK {
		t.Fatalf("a listed address = %d, want 200", rec.Code)
	}
	// The platform's accounts pass by the group the seed puts them in, not by their address.
	seeded := f.token("u", map[string]any{"email": "demo@albitor-demo.invalid", "cognito:groups": []string{platformSeededGroup}})
	if rec := get(limited, seeded, false); rec.Code != http.StatusOK {
		t.Fatalf("a seeded account = %d, want 200", rec.Code)
	}
	if rec := get(limited, with("demo@albitor-demo.invalid"), false); rec.Code != http.StatusForbidden {
		t.Fatalf("a seeded address outside the seeded group = %d, want 403", rec.Code)
	}
	if rec := get(gate(nil), with("direct@other.test"), false); rec.Code != http.StatusOK {
		t.Fatalf("no list = %d, want 200", rec.Code)
	}
}

// Under a domain list the token's address must be one the pool has verified: a person who moved
// their address to an in-domain one they have not confirmed is refused, and an in-domain verified
// one passes. The platform's own accounts are exempt by identity, the seeded group the deploy's
// seed adds them to, and never by address: an address on a seeded domain that is not in the group
// is held to the list like any other.
func TestTheDomainListNeedsAVerifiedAddressAndExemptsSeededIdentitiesOnly(t *testing.T) {
	f, srv := newFakeCognito(t)
	limited := composeIdentity(signIn{auth: &cognitoAuth{
		clientID: testClientID, endpoint: srv.URL + "/", client: srv.Client(), signupDomains: []string{"example.org"},
		verifier: newCognitoVerifier(f.issuer, testClientID, f.issuer+"/.well-known/jwks.json", srv.Client()),
	}})
	status := func(claims map[string]any) int {
		return authDo(t, limited, http.MethodGet, "/api/me", "", withBearer(f.token("u", claims))).Code
	}

	if got := status(map[string]any{"email": "a@example.org", "email_verified": true}); got != http.StatusOK {
		t.Errorf("a verified in-domain address = %d, want 200", got)
	}
	for name, claims := range map[string]map[string]any{
		"unverified in-domain":           {"email": "a@example.org", "email_verified": false},
		"no email_verified claim":        {"email": "a@example.org", "email_verified": nil},
		"email_verified as a bad string": {"email": "a@example.org", "email_verified": "yes"},
		"verified, out of domain":        {"email": "a@other.test", "email_verified": true},
		// The seeded domains buy nothing without the group, whether or not the address is verified.
		"seeded domain, no group, verified":   {"email": "demo@albitor-demo.invalid", "email_verified": true},
		"seeded domain, no group, unverified": {"email": "verifier@albitor-verify.invalid", "email_verified": false},
		"seeded role domain, other group":     {"email": "x@albitor-role.invalid", "email_verified": true, "cognito:groups": []string{"staff"}},
	} {
		if got := status(claims); got != http.StatusForbidden {
			t.Errorf("%s = %d, want 403", name, got)
		}
	}
	// The seeded identity passes by its group, whatever address it carries.
	for name, claims := range map[string]map[string]any{
		"seeded account":              {"email": "demo@albitor-demo.invalid", "email_verified": true, "cognito:groups": []string{platformSeededGroup}},
		"seeded account and its role": {"email": "admin@albitor-role.invalid", "email_verified": true, "cognito:groups": []string{"admin", platformSeededGroup}},
	} {
		if got := status(claims); got != http.StatusOK {
			t.Errorf("%s = %d, want 200", name, got)
		}
	}
}
