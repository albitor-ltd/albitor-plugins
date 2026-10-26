package shell

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"webapp/internal/identity"
)

// These tests hold the hosted sign-in flow to its contract: the state and the PKCE verifier, the
// callback's order of checks (state, code, ID token, verified email, domain rule, then a session),
// the cookies' attributes, sign-out and account deletion.

// startSignIn runs GET /api/auth/login?next= and returns the transient cookies it set, the state
// and code_challenge it sent to the hosted page.
func startSignIn(t *testing.T, h http.Handler, next string) (cookies []*http.Cookie, state, challenge string, rec *httptest.ResponseRecorder) {
	t.Helper()
	rec = authDo(t, h, http.MethodGet, "/api/auth/login?next="+url.QueryEscape(next), "", nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("GET /api/auth/login = %d, want 302", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return rec.Result().Cookies(), loc.Query().Get("state"), loc.Query().Get("code_challenge"), rec
}

// callbackWith runs GET /api/auth/callback with the given query and cookies.
func callbackWith(t *testing.T, h http.Handler, query string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return authDo(t, h, http.MethodGet, "/api/auth/callback?"+query, "", withCookies(cookies...))
}

// signedIn answers the token endpoint with a good ID token for sub, and returns the handler and fake.
func signedIn(t *testing.T, claims map[string]any) (http.Handler, *fakeCognito) {
	t.Helper()
	f := newFakeCognito(t)
	f.answerToken(func(form url.Values) (int, any) {
		if form.Get("grant_type") != "authorization_code" || form.Get("code") != "good-code" {
			return http.StatusBadRequest, map[string]string{"error": "invalid_grant"}
		}
		return http.StatusOK, map[string]string{
			"id_token": f.idToken("user-a", claims), "access_token": f.accessToken("user-a", "openid email"), "refresh_token": "refresh-a",
		}
	})
	return gateFor(f), f
}

func TestLoginRedirectsToTheHostedPageWithAStateAndAnS256Challenge(t *testing.T) {
	f := newFakeCognito(t)
	h := gateFor(f)
	cookies, state, challenge, rec := startSignIn(t, h, "/reports?tab=2")

	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, f.auth().loginURL+"&state=") {
		t.Fatalf("Location = %q, want the deploy's login URL, untouched, with the parameters appended", loc)
	}
	// The scope is the deploy's, exactly as written, never one this plugin chose.
	if !strings.Contains(loc, "scope=openid%20email%20aws.cognito.signin.user.admin") || !strings.Contains(loc, "code_challenge_method=S256") {
		t.Fatalf("Location = %q", loc)
	}
	if len(state) < 32 || len(challenge) != 43 {
		t.Fatalf("state %q and challenge %q are too short to be random S256 values", state, challenge)
	}

	byName := map[string]*http.Cookie{}
	for _, c := range cookies {
		byName[c.Name] = c
	}
	for _, name := range []string{stateCookie, verifierCookie, nextCookie} {
		c := byName[name]
		if c == nil {
			t.Fatalf("no %s cookie", name)
		}
		// Lax, not Strict: the callback is a cross-site top-level navigation, which carries Lax cookies.
		if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge <= 0 || c.MaxAge > 600 {
			t.Errorf("%s cookie = %+v, want HttpOnly, Secure, Lax, path /, max-age of minutes", name, c)
		}
	}
	if byName[stateCookie].Value != state {
		t.Errorf("the state cookie %q is not the state sent to Cognito %q", byName[stateCookie].Value, state)
	}
	verifier, _, _ := strings.Cut(byName[verifierCookie].Value, ".")
	if pkceChallenge(verifier) != challenge {
		t.Errorf("code_challenge is not the S256 of the verifier cookie")
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("the login redirect must not be cached")
	}

	// Two sign-ins never share a state or a verifier.
	_, state2, challenge2, _ := startSignIn(t, h, "/")
	if state2 == state || challenge2 == challenge {
		t.Error("two sign-ins reused a state or verifier")
	}
}

func TestLoginOnlyEverRemembersAPathOnThisSite(t *testing.T) {
	h := gateFor(newFakeCognito(t))
	for next, want := range map[string]string{
		"/reports?tab=2#top":            "/reports?tab=2#top",
		"/":                             "/",
		"":                              "/",
		"//evil.test":                   "/",
		"//evil.test/path":              "/",
		"/\\evil.test":                  "/",
		"\\\\evil.test":                 "/",
		"https://evil.test/":            "/",
		"http://evil.test":              "/",
		"javascript:alert(1)":           "/",
		"reports":                       "/",
		"/\t/evil.test":                 "/",
		"/%0d%0aSet-Cookie:x=y":         "/%0d%0aSet-Cookie:x=y", // percent-encoded: inert, still a path on this site
		"/api/auth/logout":              "/",
		"/api/me":                       "/",
		"/sign-in?next=/":               "/",
		"/sign-out":                     "/",
		"/not-allowed":                  "/",
		"/" + strings.Repeat("a", 2100): "/",
	} {
		cookies, _, _, _ := startSignIn(t, h, next)
		var got string
		for _, c := range cookies {
			if c.Name == nextCookie {
				b, err := base64.RawURLEncoding.DecodeString(c.Value)
				if err != nil {
					t.Fatalf("next %q: the next cookie is not base64url: %v", next, err)
				}
				got = string(b)
			}
		}
		if got != want {
			t.Errorf("next %q remembered as %q, want %q", next, got, want)
		}
	}
	// A raw control character cannot be sent in a URL, so it is tried at the function.
	for _, next := range []string{"/a\nb", "/a\x00b", "/a\x7fb"} {
		if got := safeNext(next); got != "/" {
			t.Errorf("safeNext(%q) = %q, want /", next, got)
		}
	}
}

func TestCallbackSignsInAndNavigatesWithoutARedirect(t *testing.T) {
	h, f := signedIn(t, nil)
	cookies, state, challenge, _ := startSignIn(t, h, "/reports?tab=2&x=<y>")

	rec := callbackWith(t, h, "code=good-code&state="+state, cookies)

	// THE STRICT-COOKIE RISK. The session cookies are SameSite=Strict and this request is a
	// cross-site top-level navigation from Cognito. A browser stores a Strict cookie set on the
	// response, but does not send it on any request whose navigation chain began on another site,
	// and that includes the redirect a 302 from here would make to the page. So the answer is a 200
	// that navigates itself: the next request is started by a page of this site and is same-site.
	if rec.Code != http.StatusOK {
		t.Fatalf("callback = %d, want 200 (a 3xx would make the first authenticated request cross-site)", rec.Code)
	}
	if rec.Header().Get("Location") != "" {
		t.Fatalf("callback sent a Location header %q: that is a redirect", rec.Header().Get("Location"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<meta http-equiv="refresh" content="0;url=/reports?tab=2&amp;x=&lt;y&gt;">`) {
		t.Errorf("the page does not navigate to next, escaped: %s", body)
	}
	if !strings.Contains(body, `<a href="/reports?tab=2&amp;x=&lt;y&gt;">`) {
		t.Errorf("the page has no link fallback to next: %s", body)
	}
	if strings.Contains(strings.ToLower(body), "<script") {
		t.Errorf("the page must not need script (the app's CSP may forbid it): %s", body)
	}
	for _, h := range []string{"Cache-Control", "Referrer-Policy"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("no %s header on the callback page", h)
		}
	}

	// The session cookies, with exactly the attributes the app depends on.
	session, refresh := cookieNamed(rec, sessionCookie), cookieNamed(rec, refreshCookie)
	if session == nil || refresh == nil {
		t.Fatalf("session cookies not set: %v", rec.Result().Cookies())
	}
	for _, c := range []*http.Cookie{session, refresh} {
		if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.MaxAge <= 0 {
			t.Errorf("%s cookie = %+v, want HttpOnly, Secure, SameSite=Strict, with a max-age", c.Name, c)
		}
	}
	if session.Path != "/" || refresh.Path != "/" || refresh.Value != "refresh-a" {
		t.Errorf("session path %q, refresh path %q value %q", session.Path, refresh.Path, refresh.Value)
	}
	// The transient cookies are cleared, with the attributes they were set with.
	for _, name := range []string{stateCookie, verifierCookie, nextCookie} {
		if c := cookieNamed(rec, name); c == nil || c.MaxAge >= 0 || c.Path != "/" {
			t.Errorf("%s cookie after the callback = %+v, want cleared", name, c)
		}
	}

	// The first authenticated request carries them: they open /api/me.
	me := authDo(t, h, http.MethodGet, "/api/me", "", withCookies(session))
	if me.Code != http.StatusOK || authDecode(t, me)["sub"] != "user-a" {
		t.Fatalf("GET /api/me with the new session = %d %s", me.Code, me.Body.String())
	}

	// What went to the token endpoint: the code, this sign-in's verifier, the deploy's redirect URI,
	// the public client, and no secret.
	call := f.callsTo("/oauth2/token")[0]
	verifier := call.form.Get("code_verifier")
	if call.form.Get("grant_type") != "authorization_code" || call.form.Get("client_id") != testClientID ||
		call.form.Get("code") != "good-code" || call.form.Get("redirect_uri") != testRedirectURI ||
		pkceChallenge(verifier) != challenge || call.header.Get("Authorization") != "" {
		t.Errorf("token request = %v (auth header %q); want the code exchange for challenge %s", call.form, call.header.Get("Authorization"), challenge)
	}
}

func TestCallbackWithoutANextCookieGoesHome(t *testing.T) {
	h, _ := signedIn(t, nil)
	cookies, state, _, _ := startSignIn(t, h, "")
	rec := callbackWith(t, h, "code=good-code&state="+state, cookies)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `content="0;url=/"`) {
		t.Fatalf("callback = %d %s, want a navigation to /", rec.Code, rec.Body.String())
	}
	// A tampered next cookie is checked again, not trusted.
	for i, c := range cookies {
		if c.Name == nextCookie {
			cookies[i] = &http.Cookie{Name: nextCookie, Value: base64.RawURLEncoding.EncodeToString([]byte("//evil.test"))}
		}
	}
	_, state, _, _ = startSignIn(t, h, "/")
	rec = callbackWith(t, h, "code=good-code&state="+state, cookies[:0:0])
	if loc := rec.Header().Get("Location"); rec.Code != http.StatusFound || loc != "/sign-in?error=expired" {
		t.Fatalf("callback with no cookies = %d %q", rec.Code, loc)
	}
}

// refusedSignIn asserts a callback ended in a redirect to the sign-in page with code, set no
// session, cleared the transient cookies and (when exchanged is false) never sent the code anywhere.
func refusedSignIn(t *testing.T, rec *httptest.ResponseRecorder, f *fakeCognito, code string, exchanged bool) {
	t.Helper()
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/sign-in?error="+code {
		t.Fatalf("callback = %d %q, want a redirect to /sign-in?error=%s", rec.Code, rec.Header().Get("Location"), code)
	}
	for _, name := range []string{sessionCookie, refreshCookie} {
		if c := cookieNamed(rec, name); c != nil && c.MaxAge > 0 {
			t.Errorf("the refused sign-in set a %s cookie", name)
		}
	}
	for _, name := range []string{stateCookie, verifierCookie, nextCookie} {
		if c := cookieNamed(rec, name); c == nil || c.MaxAge >= 0 {
			t.Errorf("the transient %s cookie was not cleared", name)
		}
	}
	if got := len(f.callsTo("/oauth2/token")) > 0; got != exchanged {
		t.Errorf("code exchanged = %v, want %v", got, exchanged)
	}
}

func TestCallbackRefusesAStateThatIsNotThisBrowsers(t *testing.T) {
	h, f := signedIn(t, nil)
	cookies, state, _, _ := startSignIn(t, h, "/")
	for name, query := range map[string]string{
		"another state":   "code=good-code&state=" + state + "x",
		"no state":        "code=good-code",
		"an empty state":  "code=good-code&state=",
		"a shorter state": "code=good-code&state=" + state[:10],
	} {
		t.Run(name, func(t *testing.T) {
			refusedSignIn(t, callbackWith(t, h, query, cookies), f, "failed", false)
		})
	}
	t.Run("no code", func(t *testing.T) {
		refusedSignIn(t, callbackWith(t, h, "state="+state, cookies), f, "failed", false)
	})
	t.Run("Cognito reporting an error", func(t *testing.T) {
		refusedSignIn(t, callbackWith(t, h, "error=access_denied&state="+state, cookies), f, "denied", false)
	})
}

func TestCallbackRefusesAMissingOrExpiredVerifier(t *testing.T) {
	h, f := signedIn(t, nil)
	cookies, state, _, _ := startSignIn(t, h, "/")
	q := "code=good-code&state=" + state
	without := func(name string) []*http.Cookie {
		var out []*http.Cookie
		for _, c := range cookies {
			if c.Name != name {
				out = append(out, c)
			}
		}
		return out
	}
	t.Run("no verifier cookie", func(t *testing.T) {
		refusedSignIn(t, callbackWith(t, h, q, without(verifierCookie)), f, "expired", false)
	})
	t.Run("no state cookie", func(t *testing.T) { refusedSignIn(t, callbackWith(t, h, q, without(stateCookie)), f, "expired", false) })
	t.Run("a verifier with no issue time", func(t *testing.T) {
		bad := append(without(verifierCookie), &http.Cookie{Name: verifierCookie, Value: "abcdef"})
		refusedSignIn(t, callbackWith(t, h, q, bad), f, "expired", false)
	})
	t.Run("a verifier older than the sign-in allows, whatever the browser kept", func(t *testing.T) {
		a := f.auth()
		later := time.Now().Add(transientMaxAge + time.Minute)
		a.now = func() time.Time { return later }
		late := composeIdentity(signIn{auth: a})
		refusedSignIn(t, callbackWith(t, late, q, cookies), f, "expired", false)
	})
	t.Run("control: the same cookies in time", func(t *testing.T) {
		if rec := callbackWith(t, h, q, cookies); rec.Code != http.StatusOK {
			t.Fatalf("callback = %d, want 200", rec.Code)
		}
	})
}

func TestCallbackRefusesATokenThePoolShouldNotHaveIssued(t *testing.T) {
	other := newTestKey(t)
	for name, claims := range map[string]map[string]any{
		"another issuer":   {"iss": "https://evil.test/pool"},
		"another audience": {"aud": "other-client"},
		"an expired token": {"exp": time.Now().Add(-time.Minute).Unix()},
		"an access token":  {"token_use": "access", "aud": nil, "client_id": testClientID},
		"no subject":       {"sub": ""},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakeCognito(t)
			f.answerToken(func(url.Values) (int, any) {
				c := f.claims("user-a", claims)
				for k, v := range c {
					if v == nil {
						delete(c, k)
					}
				}
				return http.StatusOK, map[string]string{"id_token": signToken(t, f.key, testKid, "RS256", c)}
			})
			h := gateFor(f)
			cookies, state, _, _ := startSignIn(t, h, "/")
			refusedSignIn(t, callbackWith(t, h, "code=good-code&state="+state, cookies), f, "failed", true)
		})
	}
	t.Run("a signature from another key", func(t *testing.T) {
		f := newFakeCognito(t)
		f.answerToken(func(url.Values) (int, any) {
			return http.StatusOK, map[string]string{"id_token": signToken(t, other, testKid, "RS256", f.claims("user-a", nil))}
		})
		h := gateFor(f)
		cookies, state, _, _ := startSignIn(t, h, "/")
		refusedSignIn(t, callbackWith(t, h, "code=good-code&state="+state, cookies), f, "failed", true)
	})
	t.Run("an answer with no ID token", func(t *testing.T) {
		f := newFakeCognito(t)
		f.answerToken(func(url.Values) (int, any) { return http.StatusOK, map[string]string{"access_token": "x"} })
		h := gateFor(f)
		cookies, state, _, _ := startSignIn(t, h, "/")
		refusedSignIn(t, callbackWith(t, h, "code=good-code&state="+state, cookies), f, "unavailable", true)
	})
}

func TestCallbackReportsAnEndpointThatRefusesOrIsDown(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		want   string
	}{
		"a code the endpoint refuses": {http.StatusBadRequest, "failed"},
		"an endpoint that is down":    {http.StatusBadGateway, "unavailable"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakeCognito(t)
			f.answerToken(func(url.Values) (int, any) { return tc.status, map[string]string{"error": "invalid_grant"} })
			h := gateFor(f)
			cookies, state, _, _ := startSignIn(t, h, "/")
			refusedSignIn(t, callbackWith(t, h, "code=good-code&state="+state, cookies), f, tc.want, true)
		})
	}
}

func TestCallbackRequiresAVerifiedEmail(t *testing.T) {
	for name, claims := range map[string]map[string]any{
		"email_verified false":                {"email_verified": false},
		"email_verified the string \"false\"": {"email_verified": "false"},
		"no email_verified":                   {"email_verified": nil},
		"no email":                            {"email": nil},
	} {
		t.Run(name, func(t *testing.T) {
			h, f := signedIn(t, claims)
			cookies, state, _, _ := startSignIn(t, h, "/")
			refusedSignIn(t, callbackWith(t, h, "code=good-code&state="+state, cookies), f, "unverified", true)
		})
	}
	t.Run("email_verified the string \"true\" is accepted, as Cognito has sent it", func(t *testing.T) {
		h, _ := signedIn(t, map[string]any{"email_verified": "true"})
		cookies, state, _, _ := startSignIn(t, h, "/")
		if rec := callbackWith(t, h, "code=good-code&state="+state, cookies); rec.Code != http.StatusOK {
			t.Fatalf("callback = %d, want 200", rec.Code)
		}
	})
}

func TestTheSignupDomainRuleRunsAfterTheEmailIsVerified(t *testing.T) {
	setup := func(t *testing.T, domains string, claims map[string]any) (http.Handler, *fakeCognito) {
		t.Helper()
		f := newFakeCognito(t)
		f.answerToken(func(url.Values) (int, any) {
			return http.StatusOK, map[string]string{"id_token": f.idToken("user-a", claims), "refresh_token": "refresh-a"}
		})
		a := f.auth()
		parsed, err := parseSignupDomains(domains)
		if err != nil {
			t.Fatal(err)
		}
		a.domains = parsed
		return composeIdentity(signIn{auth: a}), f
	}

	t.Run("a refused address goes to the refusal page and holds no session", func(t *testing.T) {
		h, _ := setup(t, "council.gov.uk", map[string]any{"email": "Sam.Lee@gmail.com"})
		cookies, state, _, _ := startSignIn(t, h, "/reports")
		rec := callbackWith(t, h, "code=good-code&state="+state, cookies)
		if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/not-allowed" {
			t.Fatalf("callback = %d %q, want a redirect to the refusal page", rec.Code, rec.Header().Get("Location"))
		}
		for _, name := range []string{sessionCookie, refreshCookie} {
			if c := cookieNamed(rec, name); c != nil && c.MaxAge > 0 {
				t.Errorf("a refused address was given a %s cookie", name)
			}
		}
		if c := cookieNamed(rec, stateCookie); c == nil || c.MaxAge >= 0 {
			t.Error("the transient cookies were not cleared on a refusal")
		}
	})
	t.Run("a refused address that is also unverified is unverified first", func(t *testing.T) {
		h, f := setup(t, "council.gov.uk", map[string]any{"email": "x@gmail.com", "email_verified": false})
		cookies, state, _, _ := startSignIn(t, h, "/")
		refusedSignIn(t, callbackWith(t, h, "code=good-code&state="+state, cookies), f, "unverified", true)
	})
	for name, tc := range map[string]struct{ domains, email string }{
		"an exact domain":      {"council.gov.uk", "sam@council.gov.uk"},
		"any case":             {"Council.GOV.uk", "SAM@COUNCIL.gov.uk"},
		"one of several":       {"a.example, council.gov.uk", "sam@council.gov.uk"},
		"a leading @":          {"@council.gov.uk", "sam@council.gov.uk"},
		"a subdomain wildcard": {"*.gov.uk", "sam@council.gov.uk"},
		"no rule at all":       {"", "sam@gmail.com"},
	} {
		t.Run("allows "+name, func(t *testing.T) {
			h, _ := setup(t, tc.domains, map[string]any{"email": tc.email})
			cookies, state, _, _ := startSignIn(t, h, "/")
			if rec := callbackWith(t, h, "code=good-code&state="+state, cookies); rec.Code != http.StatusOK || cookieNamed(rec, sessionCookie) == nil {
				t.Fatalf("callback = %d, want a session", rec.Code)
			}
		})
	}
	for name, tc := range map[string]struct{ domains, email string }{
		"another domain":                 {"council.gov.uk", "sam@gmail.com"},
		"a subdomain of an exact domain": {"council.gov.uk", "sam@evil.council.gov.uk"},
		"a lookalike suffix":             {"council.gov.uk", "sam@notcouncil.gov.uk"},
		"a lookalike prefix":             {"council.gov.uk", "sam@council.gov.uk.evil.test"},
		"the bare domain of a wildcard":  {"*.gov.uk", "sam@gov.uk"},
		"a lookalike of a wildcard":      {"*.gov.uk", "sam@evilgov.uk"},
	} {
		t.Run("refuses "+name, func(t *testing.T) {
			h, _ := setup(t, tc.domains, map[string]any{"email": tc.email})
			cookies, state, _, _ := startSignIn(t, h, "/")
			if rec := callbackWith(t, h, "code=good-code&state="+state, cookies); rec.Code != http.StatusFound || !strings.HasPrefix(rec.Header().Get("Location"), "/not-allowed") {
				t.Fatalf("callback = %d %q, want the refusal page", rec.Code, rec.Header().Get("Location"))
			}
		})
	}
	for name, email := range map[string]string{
		"the verifier account": "verifier@albitor-verify.invalid",
		"the demo account":     "Demo@Albitor-Demo.invalid",
	} {
		t.Run("always allows "+name+", whatever the rule says", func(t *testing.T) {
			h, _ := setup(t, "northgate.gov.uk", map[string]any{"email": email})
			cookies, state, _, _ := startSignIn(t, h, "/")
			if rec := callbackWith(t, h, "code=good-code&state="+state, cookies); rec.Code != http.StatusOK || cookieNamed(rec, sessionCookie) == nil {
				t.Fatalf("callback = %d, want a session", rec.Code)
			}
		})
	}
	t.Run("the seeded domains are exact, and still need a verified address", func(t *testing.T) {
		a := &hostedAuth{domains: []string{"northgate.gov.uk"}}
		for _, email := range []string{
			"x@evil.albitor-demo.invalid", "x@albitor-demo.invalid.evil.test", "albitor-demo.invalid",
			"x@notalbitor-verify.invalid", "x@albitor-demo.invalid@evil.test",
			"albitor-demo.invalid@evil.test", " x@albitor-demo.invalid.evil.test ",
		} {
			if err := a.acceptIDToken(jwtClaims{TokenUse: "id", Email: email, EmailVerified: true}); !errors.Is(err, errEmailNotAllowed) {
				t.Errorf("%q = %v, want errEmailNotAllowed", email, err)
			}
		}
		// The address is trimmed once, so padding does not hide a seeded address either.
		if err := a.acceptIDToken(jwtClaims{TokenUse: "id", Email: "  Demo@albitor-demo.invalid\t", EmailVerified: true}); err != nil {
			t.Errorf("padded seeded address = %v, want accepted", err)
		}
		if err := a.acceptIDToken(jwtClaims{TokenUse: "id", Email: "demo@albitor-demo.invalid"}); !errors.Is(err, errEmailUnverified) {
			t.Errorf("unverified seeded address = %v, want errEmailUnverified", err)
		}
	})
	t.Run("a seeded address passes on every request and on refresh, whatever the rule says", func(t *testing.T) {
		f := newFakeCognito(t)
		a := f.auth()
		a.domains = []string{"northgate.gov.uk"}
		h := composeIdentity(signIn{auth: a})
		seeded := f.idToken("user-a", map[string]any{"email": "demo@albitor-demo.invalid"})
		other := f.idToken("user-a", map[string]any{"email": "sam@gmail.com"})
		bearer := func(tok string) func(*http.Request) {
			return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+tok) }
		}
		if rec := authDo(t, h, http.MethodGet, "/api/items", "", bearer(seeded)); rec.Code != http.StatusOK {
			t.Errorf("bearer, seeded = %d, want 200", rec.Code)
		}
		if rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: seeded})); rec.Code != http.StatusOK {
			t.Errorf("cookie, seeded = %d, want 200", rec.Code)
		}
		// Controls: the rule still bites an ordinary address on both paths.
		if rec := authDo(t, h, http.MethodGet, "/api/items", "", bearer(other)); rec.Code != http.StatusUnauthorized {
			t.Errorf("bearer, other = %d, want 401", rec.Code)
		}
		if rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(&http.Cookie{Name: sessionCookie, Value: other})); rec.Code != http.StatusUnauthorized {
			t.Errorf("cookie, other = %d, want 401", rec.Code)
		}
		f.answerToken(func(url.Values) (int, any) { return http.StatusOK, map[string]string{"id_token": seeded} })
		expired := f.idToken("user-a", map[string]any{"exp": time.Now().Add(-time.Minute).Unix(), "email": "demo@albitor-demo.invalid"})
		rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(
			&http.Cookie{Name: sessionCookie, Value: expired}, &http.Cookie{Name: refreshCookie, Value: "refresh-a"}))
		if rec.Code != http.StatusOK {
			t.Errorf("refresh to a seeded address = %d, want 200", rec.Code)
		}
	})
	t.Run("a refresh is checked against the rule again, and a refusal ends the session", func(t *testing.T) {
		f := newFakeCognito(t)
		a := f.auth()
		a.domains = []string{"council.gov.uk"}
		changed := f.idToken("user-a", map[string]any{"email": "sam@gmail.com"})
		f.answerToken(func(url.Values) (int, any) { return http.StatusOK, map[string]string{"id_token": changed} })
		expired := f.idToken("user-a", map[string]any{"exp": time.Now().Add(-time.Minute).Unix(), "email": "sam@council.gov.uk"})
		rec := authDo(t, composeIdentity(signIn{auth: a}), http.MethodGet, "/api/items", "", withCookies(
			&http.Cookie{Name: sessionCookie, Value: expired}, &http.Cookie{Name: refreshCookie, Value: "refresh-a"}))
		body := authDecode(t, rec)
		if rec.Code != http.StatusUnauthorized || body["code"] != "email_not_allowed" {
			t.Fatalf("refresh to a refused address = %d %s, want 401 email_not_allowed", rec.Code, rec.Body.String())
		}
		if c := cookieNamed(rec, sessionCookie); c == nil || c.MaxAge >= 0 {
			t.Errorf("the session cookie was not cleared: %v", c)
		}
		// Control: an allowed address refreshes.
		f.answerToken(func(url.Values) (int, any) {
			return http.StatusOK, map[string]string{"id_token": f.idToken("user-a", map[string]any{"email": "sam@council.gov.uk"})}
		})
		rec = authDo(t, composeIdentity(signIn{auth: a}), http.MethodGet, "/api/items", "", withCookies(
			&http.Cookie{Name: sessionCookie, Value: expired}, &http.Cookie{Name: refreshCookie, Value: "refresh-a"}))
		if rec.Code != http.StatusOK {
			t.Fatalf("refresh to an allowed address = %d", rec.Code)
		}
	})
}

func TestTheSignupDomainRuleIsParsedStrictly(t *testing.T) {
	for _, ok := range []string{"", "  ", "example.org", "a.example.org,b.example.org", "*.example.org", "@example.org", "example.org example.com", "EXAMPLE.ORG"} {
		if _, err := parseSignupDomains(ok); err != nil {
			t.Errorf("parseSignupDomains(%q) = %v, want ok", ok, err)
		}
	}
	for _, bad := range []string{"org", "*.org", "*", "example", "exa mple.org!", "example..org", "-example.org", "http://example.org", "user@example.org", "*example.org", "example.org/"} {
		if _, err := parseSignupDomains(bad); err == nil {
			t.Errorf("parseSignupDomains(%q) accepted a malformed entry", bad)
		}
	}
	if signupAllowed([]string{"example.org"}, "no-at-sign") || signupAllowed([]string{"example.org"}, "@example.org") || signupAllowed([]string{"example.org"}, "a@") {
		t.Error("an address with no local part or no domain was allowed")
	}
}

func TestSignOutRevokesTheRefreshTokenAndReturnsTheLogoutURL(t *testing.T) {
	f := newFakeCognito(t)
	f.revoke = func(url.Values) int { return http.StatusOK }
	h := gateFor(f)
	cookies := []*http.Cookie{{Name: sessionCookie, Value: f.idToken("user-a", nil)}, {Name: refreshCookie, Value: "refresh-a"}}

	rec := authDo(t, h, http.MethodPost, "/api/auth/sign-out", "{}", withCookies(cookies...))
	if rec.Code != http.StatusOK || authDecode(t, rec)["logout_url"] != testLogoutURL {
		t.Fatalf("sign-out = %d %s, want 200 with the logout URL", rec.Code, rec.Body.String())
	}
	calls := f.callsTo("/oauth2/revoke")
	if len(calls) != 1 || calls[0].form.Get("token") != "refresh-a" || calls[0].form.Get("client_id") != testClientID {
		t.Fatalf("revoke calls = %+v, want one for the refresh token", calls)
	}
	for _, name := range []string{sessionCookie, refreshCookie} {
		if c := cookieNamed(rec, name); c == nil || c.MaxAge >= 0 || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode {
			t.Errorf("%s cookie after sign-out = %+v, want cleared with the same attributes", name, c)
		}
	}
	if c := cookieNamed(rec, refreshCookie); c.Path != "/" || c.Domain != "" || !strings.HasPrefix(c.Name, "__Host-") {
		t.Errorf("the refresh cookie %+v is not a __Host- cookie: Path=/, no Domain", c)
	}

	t.Run("a revoke that fails still signs the person out", func(t *testing.T) {
		f.revoke = func(url.Values) int { return http.StatusBadGateway }
		rec := authDo(t, h, http.MethodPost, "/api/auth/sign-out", "{}", withCookies(cookies...))
		if c := cookieNamed(rec, sessionCookie); rec.Code != http.StatusOK || c == nil || c.MaxAge >= 0 {
			t.Fatalf("sign-out with a failing revoke = %d, session cookie %v", rec.Code, c)
		}
	})
	t.Run("with no session there is nothing to revoke", func(t *testing.T) {
		before := len(f.callsTo("/oauth2/revoke"))
		rec := authDo(t, h, http.MethodPost, "/api/auth/sign-out", "{}", nil)
		if rec.Code != http.StatusOK || len(f.callsTo("/oauth2/revoke")) != before {
			t.Fatalf("sign-out with no session = %d, revokes %d -> %d", rec.Code, before, len(f.callsTo("/oauth2/revoke")))
		}
	})
}

func TestThePostRoutesAreCSRFSafe(t *testing.T) {
	f := newFakeCognito(t)
	h := gateFor(f)
	session := withCookies(&http.Cookie{Name: sessionCookie, Value: f.idToken("user-a", nil)}, &http.Cookie{Name: refreshCookie, Value: "refresh-a"})
	for _, path := range []string{"/api/auth/sign-out", "/api/auth/delete"} {
		for _, ct := range []string{"application/x-www-form-urlencoded", "text/plain", "multipart/form-data; boundary=x", ""} {
			rec := authDo(t, h, http.MethodPost, path, "a=b", func(r *http.Request) {
				session(r)
				r.Header.Set("Content-Type", ct)
			})
			if rec.Code != http.StatusUnsupportedMediaType {
				t.Errorf("POST %s as %q = %d, want 415: a form from another site must not sign anyone out or delete anything", path, ct, rec.Code)
			}
		}
		if rec := authDo(t, h, http.MethodGet, path, "", session); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, rec.Code)
		}
	}
	if len(f.callsTo("/oauth2/revoke"))+len(f.callsTo("/#DeleteUser")) != 0 {
		t.Error("a refused request still reached Cognito")
	}
	if rec := authDo(t, h, http.MethodPost, "/api/auth/login", "{}", nil); rec.Code != http.StatusNotFound {
		t.Errorf("POST /api/auth/login = %d, want 404", rec.Code)
	}
	if rec := authDo(t, h, http.MethodPost, "/api/auth/sign-in", `{"email":"a@b.c","password":"x"}`, nil); rec.Code != http.StatusNotFound {
		t.Errorf("POST /api/auth/sign-in = %d, want 404: there is no password sign-in", rec.Code)
	}
}

// deletion sets up a signed-in person whose refresh yields an access token with scope.
func deletion(t *testing.T, scope string) (http.Handler, *fakeCognito, []*http.Cookie) {
	t.Helper()
	f := newFakeCognito(t)
	f.answerToken(func(form url.Values) (int, any) {
		if form.Get("grant_type") != "refresh_token" || form.Get("refresh_token") != "refresh-a" {
			return http.StatusBadRequest, map[string]string{"error": "invalid_grant"}
		}
		return http.StatusOK, map[string]string{"id_token": f.idToken("user-a", nil), "access_token": f.accessToken("user-a", scope)}
	})
	f.deleteUser = func(map[string]any) (int, any) { return http.StatusOK, map[string]string{} }
	cookies := []*http.Cookie{{Name: sessionCookie, Value: f.idToken("user-a", nil)}, {Name: refreshCookie, Value: "refresh-a"}}
	return gateFor(f), f, cookies
}

func TestDeleteRemovesTheAccountWithTheAccessToken(t *testing.T) {
	h, f, cookies := deletion(t, "openid email "+adminScope)
	var hooked []identity.Identity
	OnAccountDeleted = func(_ context.Context, id identity.Identity) error {
		hooked = append(hooked, id)
		return nil
	}
	t.Cleanup(func() { OnAccountDeleted = nil })

	rec := authDo(t, h, http.MethodPost, "/api/auth/delete", "{}", withCookies(cookies...))
	if rec.Code != http.StatusOK || authDecode(t, rec)["logout_url"] != testLogoutURL {
		t.Fatalf("delete = %d %s, want 200 with the logout URL", rec.Code, rec.Body.String())
	}
	calls := f.callsTo("/#DeleteUser")
	if len(calls) != 1 {
		t.Fatalf("DeleteUser calls = %+v, want one", calls)
	}
	if got, _ := calls[0].json["AccessToken"].(string); strings.Count(got, ".") != 2 {
		t.Fatalf("DeleteUser was not given an access token: %q", got)
	}
	if len(hooked) != 1 || hooked[0].Sub != "user-a" {
		t.Fatalf("OnAccountDeleted calls = %+v, want one for user-a", hooked)
	}
	for _, name := range []string{sessionCookie, refreshCookie} {
		if c := cookieNamed(rec, name); c == nil || c.MaxAge >= 0 {
			t.Errorf("%s cookie after delete = %+v, want cleared", name, c)
		}
	}
	// The app's data hook failing does not undo or hide the deletion: the person cannot retry.
	OnAccountDeleted = func(context.Context, identity.Identity) error { return errors.New("table unavailable") }
	if rec := authDo(t, h, http.MethodPost, "/api/auth/delete", "{}", withCookies(cookies...)); rec.Code != http.StatusOK {
		t.Fatalf("delete with a failing hook = %d, want 200", rec.Code)
	}
}

func TestDeleteFailsCleanlyWithoutTheAdminScope(t *testing.T) {
	h, f, cookies := deletion(t, "openid email")
	rec := authDo(t, h, http.MethodPost, "/api/auth/delete", "{}", withCookies(cookies...))
	if body := authDecode(t, rec); rec.Code != http.StatusForbidden || body["code"] != "scope_missing" || !strings.Contains(body["error"].(string), "cannot delete accounts") {
		t.Fatalf("delete without the scope = %d %s, want 403 scope_missing with a clear message", rec.Code, rec.Body.String())
	}
	if len(f.callsTo("/#DeleteUser")) != 0 {
		t.Error("DeleteUser was called with a token known to lack the scope")
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("a refused deletion changed the cookies: %v", rec.Result().Cookies())
	}

	// And Cognito's own refusal reads the same.
	h, f, cookies = deletion(t, "openid email "+adminScope)
	f.deleteUser = func(map[string]any) (int, any) {
		return http.StatusBadRequest, map[string]string{"__type": "NotAuthorizedException", "message": "Access Token does not have required scopes"}
	}
	rec = authDo(t, h, http.MethodPost, "/api/auth/delete", "{}", withCookies(cookies...))
	if rec.Code != http.StatusForbidden || authDecode(t, rec)["code"] != "scope_missing" {
		t.Fatalf("Cognito refusing the scope = %d %s, want 403 scope_missing", rec.Code, rec.Body.String())
	}
}

func TestDeleteRefusesWhoCannotProveASession(t *testing.T) {
	h, f, cookies := deletion(t, adminScope)
	t.Run("no session", func(t *testing.T) {
		if rec := authDo(t, h, http.MethodPost, "/api/auth/delete", "{}", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("delete with no session = %d, want 401", rec.Code)
		}
	})
	t.Run("a bearer token and no refresh cookie", func(t *testing.T) {
		rec := authDo(t, h, http.MethodPost, "/api/auth/delete", "{}", withBearer(f.idToken("user-a", nil)))
		if rec.Code != http.StatusUnauthorized || authDecode(t, rec)["code"] != "session_expired" {
			t.Fatalf("delete = %d %s, want 401 session_expired", rec.Code, rec.Body.String())
		}
	})
	t.Run("a revoked refresh token", func(t *testing.T) {
		stale := []*http.Cookie{cookies[0], {Name: refreshCookie, Value: "revoked"}}
		rec := authDo(t, h, http.MethodPost, "/api/auth/delete", "{}", withCookies(stale...))
		if rec.Code != http.StatusUnauthorized || authDecode(t, rec)["code"] != "session_expired" {
			t.Fatalf("delete = %d %s, want 401 session_expired", rec.Code, rec.Body.String())
		}
	})
	t.Run("an access token for somebody else", func(t *testing.T) {
		f.answerToken(func(url.Values) (int, any) {
			return http.StatusOK, map[string]string{"id_token": f.idToken("user-a", nil), "access_token": f.accessToken("user-b", adminScope)}
		})
		rec := authDo(t, h, http.MethodPost, "/api/auth/delete", "{}", withCookies(cookies...))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("delete = %d, want 401", rec.Code)
		}
	})
	if len(f.callsTo("/#DeleteUser")) != 0 {
		t.Error("DeleteUser was called for a request that could not prove a session")
	}
	t.Run("a pool that cannot be reached is not a success", func(t *testing.T) {
		h, f, cookies := deletion(t, adminScope)
		f.deleteUser = nil // the fake answers 400 with no __type
		rec := authDo(t, h, http.MethodPost, "/api/auth/delete", "{}", withCookies(cookies...))
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("delete = %d, want 502", rec.Code)
		}
		if c := cookieNamed(rec, sessionCookie); c != nil {
			t.Errorf("a failed deletion cleared the session: %v", c)
		}
	})
}

func TestNavigationPageEscapesWhatItNavigatesTo(t *testing.T) {
	rec := httptest.NewRecorder()
	writeNavigation(rec, `/a"><script>alert(1)</script>`)
	if body := rec.Body.String(); strings.Contains(body, "<script>") || strings.Contains(body, `"><`) {
		t.Fatalf("the navigation page did not escape its target: %s", body)
	}
	if !regexp.MustCompile(`content="0;url=/a&#34;&gt;`).MatchString(rec.Body.String()) {
		t.Fatalf("unexpected escaping: %s", rec.Body.String())
	}
}

func TestACookieThatCannotBeStoredIsRefusedNotDroppedSilently(t *testing.T) {
	f := newFakeCognito(t)
	f.answerToken(func(url.Values) (int, any) {
		return http.StatusOK, map[string]string{"id_token": f.idToken("user-a", map[string]any{"pad": strings.Repeat("x", 5000)}), "refresh_token": "r"}
	})
	h := gateFor(f)
	cookies, state, _, _ := startSignIn(t, h, "/")
	refusedSignIn(t, callbackWith(t, h, "code=good-code&state="+state, cookies), f, "toolarge", true)

	// And a refresh that would need one ends the session instead of looping.
	expired := f.idToken("user-a", map[string]any{"exp": time.Now().Add(-time.Minute).Unix()})
	rec := authDo(t, h, http.MethodGet, "/api/items", "", withCookies(
		&http.Cookie{Name: sessionCookie, Value: expired}, &http.Cookie{Name: refreshCookie, Value: "refresh-a"}))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh with an oversized token = %d, want 401", rec.Code)
	}
}

func TestARefreshKeepsARotatedRefreshToken(t *testing.T) {
	f := newFakeCognito(t)
	f.answerToken(func(url.Values) (int, any) {
		return http.StatusOK, map[string]string{"id_token": f.idToken("user-a", nil), "refresh_token": "refresh-b"}
	})
	expired := f.idToken("user-a", map[string]any{"exp": time.Now().Add(-time.Minute).Unix()})
	rec := authDo(t, gateFor(f), http.MethodGet, "/api/items", "", withCookies(
		&http.Cookie{Name: sessionCookie, Value: expired}, &http.Cookie{Name: refreshCookie, Value: "refresh-a"}))
	if c := cookieNamed(rec, refreshCookie); rec.Code != http.StatusOK || c == nil || c.Value != "refresh-b" || c.MaxAge <= 0 {
		t.Fatalf("refresh = %d, refresh cookie %v; want the rotated token kept", rec.Code, c)
	}
}

func TestAllTheCookiesAreHostCookies(t *testing.T) {
	f := newFakeCognito(t)
	h := gateFor(f)
	cookies, state, _, _ := startSignIn(t, h, "/")
	f.answerToken(func(url.Values) (int, any) {
		return http.StatusOK, map[string]string{"id_token": f.idToken("user-a", nil), "refresh_token": "r"}
	})
	rec := callbackWith(t, h, "code=c&state="+state, cookies)
	all := append(cookies, rec.Result().Cookies()...)
	if len(all) < 5 {
		t.Fatalf("only %d cookies seen", len(all))
	}
	for _, c := range all {
		if !strings.HasPrefix(c.Name, "__Host-") || c.Path != "/" || c.Domain != "" || !c.Secure {
			t.Errorf("cookie %+v must be a __Host- cookie: Secure, Path=/, no Domain", c)
		}
	}
}
