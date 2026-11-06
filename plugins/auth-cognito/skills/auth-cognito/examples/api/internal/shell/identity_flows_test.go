package shell

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"webapp/internal/localauth"
)

// These tests hold the sign-in calls (identity_flows.go) to their contract: an HttpOnly session
// from a sign-in, the cookies kept to their paths, sign-up and recovery reaching the pool, a
// local run answering that the calls are not available, and the body capped.

func TestSignInSetsAnHttpOnlySession(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	f.answer("InitiateAuth", func(in map[string]any) (int, any) {
		params, _ := in["AuthParameters"].(map[string]any)
		switch {
		case in["AuthFlow"] != "USER_PASSWORD_AUTH":
			return http.StatusBadRequest, map[string]string{"__type": "InvalidParameterException"}
		case params["USERNAME"] == "unconfirmed@example.invalid":
			return http.StatusBadRequest, map[string]string{"__type": "UserNotConfirmedException", "message": "User is not confirmed."}
		case params["USERNAME"] != "verifier@example.invalid" || params["PASSWORD"] != "right-one":
			return http.StatusBadRequest, map[string]string{"__type": "com.amazonaws#NotAuthorizedException", "message": "Incorrect username or password."}
		}
		return http.StatusOK, map[string]any{"AuthenticationResult": map[string]string{
			"IdToken": f.token("verifier", nil), "AccessToken": "access", "RefreshToken": "refresh-v",
		}}
	})

	rec := authDo(t, h, http.MethodPost, "/api/auth/sign-in", `{"email":" Verifier@Example.invalid ","password":"right-one"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("sign-in = %d: %s", rec.Code, rec.Body.String())
	}
	if got := authDecode(t, rec)["sub"]; got != "verifier" {
		t.Fatalf("sign-in answered sub %v", got)
	}
	if strings.Contains(rec.Body.String(), "refresh-v") || strings.Contains(rec.Body.String(), "access") {
		t.Fatalf("sign-in put a token in the body: %s", rec.Body.String())
	}
	for _, name := range []string{sessionCookie, refreshCookie} {
		c := responseCookie(rec, name)
		if c == nil || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode {
			t.Fatalf("%s cookie = %+v, want HttpOnly, Secure, SameSite=Strict", name, c)
		}
	}

	// Another site's form can post text/plain; a sign-in from one is refused.
	formPost := authDo(t, h, http.MethodPost, "/api/auth/sign-in", `{"email":"verifier@example.invalid","password":"right-one"}`,
		func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") })
	if formPost.Code != http.StatusUnsupportedMediaType || responseCookie(formPost, sessionCookie) != nil {
		t.Fatalf("a text/plain sign-in = %d, want 415 and no session", formPost.Code)
	}

	for body, want := range map[string]string{
		`{"email":"verifier@example.invalid","password":"wrong"}`:        "invalid_credentials",
		`{"email":"unconfirmed@example.invalid","password":"right-one"}`: "not_confirmed",
		`{"password":"right-one"}`:                                       "invalid_input",
	} {
		rec := authDo(t, h, http.MethodPost, "/api/auth/sign-in", body, nil)
		if got := authDecode(t, rec)["code"]; got != want {
			t.Fatalf("sign-in %s: code %v, want %s", body, got, want)
		}
		if responseCookie(rec, sessionCookie) != nil {
			t.Fatalf("a failed sign-in set a session")
		}
	}
}

func TestSignUpConfirmAndResetReachThePool(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	ok := func(map[string]any) (int, any) { return http.StatusOK, map[string]any{} }
	f.answer("SignUp", func(in map[string]any) (int, any) {
		if in["ClientId"] != testClientID || in["Username"] != "new@example.invalid" {
			return http.StatusBadRequest, map[string]string{"__type": "InvalidParameterException"}
		}
		return http.StatusOK, map[string]any{"UserConfirmed": false, "UserSub": "new"}
	})
	f.answer("ConfirmSignUp", func(in map[string]any) (int, any) {
		if in["ConfirmationCode"] != "123456" {
			return http.StatusBadRequest, map[string]string{"__type": "CodeMismatchException"}
		}
		return http.StatusOK, map[string]any{}
	})
	f.answer("ResendConfirmationCode", ok)
	f.answer("ForgotPassword", ok)
	f.answer("ConfirmForgotPassword", ok)

	steps := []struct {
		path, body string
		status     int
		code       string
	}{
		{"/api/auth/sign-up", `{"email":"new@example.invalid","password":"Long-enough-1"}`, http.StatusCreated, ""},
		{"/api/auth/confirm", `{"email":"new@example.invalid","code":"000000"}`, http.StatusBadRequest, "invalid_code"},
		{"/api/auth/confirm", `{"email":"new@example.invalid","code":"123456"}`, http.StatusNoContent, ""},
		{"/api/auth/resend", `{"email":"new@example.invalid"}`, http.StatusNoContent, ""},
		{"/api/auth/forgot", `{"email":"new@example.invalid"}`, http.StatusNoContent, ""},
		{"/api/auth/reset", `{"email":"new@example.invalid","code":"123456","password":"Long-enough-2"}`, http.StatusNoContent, ""},
		{"/api/auth/unknown", `{"email":"new@example.invalid"}`, http.StatusNotFound, ""},
	}
	for _, s := range steps {
		rec := authDo(t, h, http.MethodPost, s.path, s.body, nil)
		if rec.Code != s.status {
			t.Fatalf("POST %s = %d, want %d: %s", s.path, rec.Code, s.status, rec.Body.String())
		}
		if s.code != "" {
			if got := authDecode(t, rec)["code"]; got != s.code {
				t.Fatalf("POST %s code = %v, want %s", s.path, got, s.code)
			}
		}
	}
	if rec := authDo(t, h, http.MethodGet, "/api/auth/sign-in", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("GET /api/auth/sign-in = %d, want 404", rec.Code)
	}
}

// An allow-list refuses a sign-up from another domain with a clear message and never reaches the
// pool; sign-in and the people already in the pool are untouched, and an unset list is open.
func TestSignUpDomainAllowList(t *testing.T) {
	f, srv := newFakeCognito(t)
	reached := 0
	f.answer("SignUp", func(map[string]any) (int, any) {
		reached++
		return http.StatusOK, map[string]any{"UserConfirmed": false}
	})
	auth := func(domains []string) http.Handler {
		return composeIdentity(signIn{auth: &cognitoAuth{
			clientID: testClientID, endpoint: srv.URL + "/", client: srv.Client(), signupDomains: domains,
			verifier: newCognitoVerifier(f.issuer, testClientID, f.issuer+"/.well-known/jwks.json", srv.Client()),
		}})
	}
	signUp := func(h http.Handler, email string) *httptest.ResponseRecorder {
		return authDo(t, h, http.MethodPost, "/api/auth/sign-up", `{"email":"`+email+`","password":"Long-enough-1"}`, nil)
	}

	limited := auth([]string{"example.org"})
	rec := signUp(limited, "Outsider@other.test")
	body := authDecode(t, rec)
	if rec.Code != http.StatusForbidden || body["code"] != "email_not_allowed" || !strings.Contains(body["error"].(string), "email domains") || reached != 0 {
		t.Fatalf("outside the list = %d %v (pool reached %d times), want 403 email_not_allowed and no pool call", rec.Code, body, reached)
	}
	if rec := signUp(limited, "Person@Example.org"); rec.Code != http.StatusCreated || reached != 1 {
		t.Fatalf("inside the list = %d (pool reached %d times), want 201", rec.Code, reached)
	}
	if rec := signUp(auth(nil), "anyone@other.test"); rec.Code != http.StatusCreated || reached != 2 {
		t.Fatalf("no list = %d, want open sign-up", rec.Code)
	}
	// The list judges registering only: confirm and sign-in are not gated on it.
	f.answer("ConfirmSignUp", func(map[string]any) (int, any) { return http.StatusOK, map[string]any{} })
	if rec := authDo(t, limited, http.MethodPost, "/api/auth/confirm", `{"email":"old@other.test","code":"123456"}`, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("confirm for an existing sign-up = %d, want 204", rec.Code)
	}
}

func TestSignOutRevokesAndClears(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	f.answer("RevokeToken", func(map[string]any) (int, any) { return http.StatusOK, map[string]any{} })

	rec := authDo(t, h, http.MethodPost, "/api/auth/sign-out", "", withCookies(&http.Cookie{Name: refreshCookie, Value: "refresh-a"}))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("sign-out = %d", rec.Code)
	}
	if !f.called("RevokeToken") {
		t.Fatal("sign-out did not revoke the refresh token")
	}
	for _, name := range []string{sessionCookie, refreshCookie} {
		if c := responseCookie(rec, name); c == nil || c.MaxAge >= 0 {
			t.Fatalf("sign-out did not clear %s", name)
		}
	}
}

// The local run's acceptance check: identities signed in through the local
// issuer are different subjects to the API, with no sign-up, no e-mail and no
// call to Cognito, and the admin identity's group reaches /api/me.
func TestLocalIdentitiesSignInThroughTheLocalIssuer(t *testing.T) {
	srv, _ := startLocalRun(t)
	call := func(method, path, body, token string) (int, map[string]any) {
		req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	signIn := func(name string) string {
		status, out := call(http.MethodPost, localauth.Path+"/token", `{"identity":"`+name+`"}`, "")
		if status != http.StatusOK {
			t.Fatalf("local sign-in as %s = %d", name, status)
		}
		return out["id_token"].(string)
	}
	alice, bob, admin := signIn("alice"), signIn("bob"), signIn("admin")

	if status, _ := call(http.MethodGet, "/api/items", "", ""); status != http.StatusUnauthorized {
		t.Fatalf("GET /api/items with no token = %d, want 401", status)
	}
	_, aliceItems := call(http.MethodGet, "/api/items", "", alice)
	_, bobItems := call(http.MethodGet, "/api/items", "", bob)
	if aliceItems["owner"] == nil || aliceItems["owner"] == bobItems["owner"] {
		t.Fatalf("owners alice=%v bob=%v, want two different ones", aliceItems["owner"], bobItems["owner"])
	}
	status, me := call(http.MethodGet, "/api/me", "", admin)
	groups, _ := me["groups"].([]any)
	if status != http.StatusOK || me["email"] != "admin@example.test" || len(groups) != 1 || groups[0] != "admin" {
		t.Fatalf("GET /api/me as admin = %d %v", status, me)
	}
	if status, out := call(http.MethodPost, "/api/auth/sign-in", `{"email":"a@example.invalid","password":"x"}`, ""); status != http.StatusNotImplemented || out["code"] != "local" {
		t.Fatalf("sign-in call in a local run = %d %v, want 501 local", status, out)
	}
}

// An invited person's first sign-in: the pool answers the temporary password with
// NEW_PASSWORD_REQUIRED, the API hands the screen the challenge session, and the
// second call trades it plus their own password for the session cookies.
func TestAnInvitedPersonSetsTheirOwnPasswordAtFirstSignIn(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	f.answer("InitiateAuth", func(in map[string]any) (int, any) {
		params, _ := in["AuthParameters"].(map[string]any)
		switch params["PASSWORD"] {
		case "Temp-0nly-pw":
			return http.StatusOK, map[string]any{"ChallengeName": "NEW_PASSWORD_REQUIRED", "Session": "challenge-session"}
		case "mfa":
			return http.StatusOK, map[string]any{"ChallengeName": "SMS_MFA", "Session": "mfa-session"}
		}
		return http.StatusBadRequest, map[string]string{"__type": "NotAuthorizedException", "message": "Incorrect username or password."}
	})
	var gotResponse map[string]any
	f.answer("RespondToAuthChallenge", func(in map[string]any) (int, any) {
		gotResponse = in
		responses, _ := in["ChallengeResponses"].(map[string]any)
		switch {
		case in["Session"] != "challenge-session":
			return http.StatusBadRequest, map[string]string{"__type": "NotAuthorizedException", "message": "Invalid session for the user."}
		case responses["NEW_PASSWORD"] == "short":
			return http.StatusBadRequest, map[string]string{"__type": "InvalidPasswordException", "message": "Password did not conform with policy: Password not long enough"}
		}
		return http.StatusOK, map[string]any{"AuthenticationResult": map[string]string{
			"IdToken": f.token("invitee", nil), "AccessToken": "access", "RefreshToken": "refresh-i",
		}}
	})

	rec := authDo(t, h, http.MethodPost, "/api/auth/sign-in", `{"email":"Invitee@Example.invalid","password":"Temp-0nly-pw"}`, nil)
	out := authDecode(t, rec)
	if rec.Code != http.StatusConflict || out["code"] != "new_password_required" || out["session"] != "challenge-session" {
		t.Fatalf("sign-in with a temporary password = %d %v, want 409 new_password_required with the session", rec.Code, out)
	}
	if responseCookie(rec, sessionCookie) != nil {
		t.Fatal("a challenge set a session")
	}

	// Any other challenge stays unsupported and does not leak its session.
	rec = authDo(t, h, http.MethodPost, "/api/auth/sign-in", `{"email":"invitee@example.invalid","password":"mfa"}`, nil)
	out = authDecode(t, rec)
	if out["code"] != "challenge_unsupported" || out["session"] != nil {
		t.Fatalf("an MFA challenge = %v, want challenge_unsupported with no session", out)
	}

	rec = authDo(t, h, http.MethodPost, "/api/auth/new-password",
		`{"email":" Invitee@Example.invalid ","password":"A-brand-new-pw-1","session":"challenge-session"}`, nil)
	if rec.Code != http.StatusOK || authDecode(t, rec)["sub"] != "invitee" {
		t.Fatalf("new-password = %d: %s", rec.Code, rec.Body.String())
	}
	for _, name := range []string{sessionCookie, refreshCookie} {
		if c := responseCookie(rec, name); c == nil || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode {
			t.Fatalf("%s cookie = %+v, want HttpOnly, Secure, SameSite=Strict", name, c)
		}
	}
	responses, _ := gotResponse["ChallengeResponses"].(map[string]any)
	if gotResponse["ChallengeName"] != "NEW_PASSWORD_REQUIRED" || responses["USERNAME"] != "invitee@example.invalid" ||
		responses["NEW_PASSWORD"] != "A-brand-new-pw-1" {
		t.Fatalf("RespondToAuthChallenge got %v", gotResponse)
	}
	if strings.Contains(rec.Body.String(), "refresh-i") {
		t.Fatalf("new-password put a token in the body: %s", rec.Body.String())
	}

	for _, tc := range []struct{ body, code string }{
		{`{"email":"invitee@example.invalid","password":"short","session":"challenge-session"}`, "invalid_password"},
		{`{"email":"invitee@example.invalid","password":"A-brand-new-pw-1","session":"stale"}`, "session_expired"},
		{`{"email":"invitee@example.invalid","password":"A-brand-new-pw-1"}`, "invalid_input"},
		{`{"email":"invitee@example.invalid","session":"challenge-session"}`, "invalid_input"},
		{`{"password":"A-brand-new-pw-1","session":"challenge-session"}`, "invalid_input"},
	} {
		rec := authDo(t, h, http.MethodPost, "/api/auth/new-password", tc.body, nil)
		if got := authDecode(t, rec)["code"]; got != tc.code || responseCookie(rec, sessionCookie) != nil {
			t.Fatalf("new-password %s: code %v, want %s and no session", tc.body, got, tc.code)
		}
	}
}

// A pool that asks for the new password a second time must not loop the screen.
func TestANewPasswordChallengeThatRepeatsIsUnsupported(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	f.answer("RespondToAuthChallenge", func(map[string]any) (int, any) {
		return http.StatusOK, map[string]any{"ChallengeName": "NEW_PASSWORD_REQUIRED", "Session": "again"}
	})
	rec := authDo(t, h, http.MethodPost, "/api/auth/new-password",
		`{"email":"invitee@example.invalid","password":"A-brand-new-pw-1","session":"s"}`, nil)
	out := authDecode(t, rec)
	if rec.Code != http.StatusConflict || out["code"] != "challenge_unsupported" || out["session"] != nil {
		t.Fatalf("a repeated challenge = %d %v, want 409 challenge_unsupported", rec.Code, out)
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

func TestSignInCallsSayTheyAreNotAvailableWithoutAPool(t *testing.T) {
	h := composeIdentity(signIn{anonymous: true})
	if rec := authDo(t, h, http.MethodPost, "/api/auth/sign-in", `{}`, nil); rec.Code != http.StatusNotImplemented {
		t.Fatalf("sign-in with no pool = %d, want 501", rec.Code)
	}
	if rec := authDo(t, h, http.MethodGet, "/api/auth/config", "", nil); authDecode(t, rec)["sign_up"] != signUpOpen {
		t.Fatalf("GET /api/auth/config with no pool = %s", rec.Body.String())
	}
}

func TestSignOutFormPostIsSameSiteOnlyAndRedirects(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	f.answer("RevokeToken", func(map[string]any) (int, any) { return http.StatusOK, map[string]any{} })
	form := func(edit func(*http.Request)) func(*http.Request) {
		return func(r *http.Request) {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.AddCookie(&http.Cookie{Name: refreshCookie, Value: "refresh-a"})
			edit(r)
		}
	}

	for name, edit := range map[string]func(*http.Request){
		"no origin at all": func(*http.Request) {},
		"another origin":   func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") },
		"cross-site fetch": func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") },
	} {
		rec := authDo(t, h, http.MethodPost, "/api/auth/sign-out", "", form(edit))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s: form sign-out = %d, want 403", name, rec.Code)
		}
		if f.called("RevokeToken") || responseCookie(rec, sessionCookie) != nil {
			t.Fatalf("%s: a refused form post must not revoke or clear anything", name)
		}
	}

	for name, edit := range map[string]func(*http.Request){
		"origin names this host": func(r *http.Request) { r.Header.Set("Origin", "http://"+r.Host) },
		"browser says same-site": func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "same-origin") },
	} {
		rec := authDo(t, h, http.MethodPost, "/api/auth/sign-out", "", form(edit))
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/sign-in" {
			t.Fatalf("%s: form sign-out = %d to %q, want 303 to /sign-in", name, rec.Code, rec.Header().Get("Location"))
		}
		for _, c := range []string{sessionCookie, refreshCookie} {
			if got := responseCookie(rec, c); got == nil || got.MaxAge >= 0 {
				t.Fatalf("%s: form sign-out did not clear %s", name, c)
			}
		}
	}
	if !f.called("RevokeToken") {
		t.Fatal("form sign-out did not revoke the refresh token")
	}

	// A GET never signs anyone out.
	if rec := authDo(t, h, http.MethodGet, "/api/auth/sign-out", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("GET /api/auth/sign-out = %d, want 404", rec.Code)
	}
}

// In a local run there is no pool, but the Sign out button still has to work: a same-site form
// post clears the cookies and goes to /sign-in, never to the 501 JSON.
func TestLocalRunFormSignOutRedirects(t *testing.T) {
	srv, _ := startLocalRun(t)
	client := srv.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	post := func(origin string) *http.Response {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/sign-out", strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp
	}
	if resp := post(srv.URL); resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/sign-in" {
		t.Fatalf("local form sign-out = %d to %q, want 303 to /sign-in", resp.StatusCode, resp.Header.Get("Location"))
	}
	if resp := post("https://evil.example"); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site local form sign-out = %d, want 403", resp.StatusCode)
	}
}
