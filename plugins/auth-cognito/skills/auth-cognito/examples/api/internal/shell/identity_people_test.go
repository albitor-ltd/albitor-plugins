package shell

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode"
)

// These tests hold the invite-only half of the seam to its contract: the mode is public and
// defaults to open; invite-only has no self sign-up or recovery; a signed-in person can change
// their own password in either mode; the People admin exists only in an invite-only app, only for
// the admin group, signs its Cognito calls, and hands a temporary password back once, uncached and
// unlogged.

const (
	testAdminGroup     = "admins"
	testUserAdminGroup = "user-admins"
)

func init() {
	adminCredentials = func() awsCredentials {
		return awsCredentials{AccessKeyID: "AKID", SecretAccessKey: "s", SessionToken: "tok"}
	}
	adminClock = func() time.Time { return time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) }
}

// withAdminCredentials swaps the credentials the signed administrator calls read for one test.
func withAdminCredentials(t *testing.T, c awsCredentials) {
	t.Helper()
	prev := adminCredentials
	adminCredentials = func() awsCredentials { return c }
	t.Cleanup(func() { adminCredentials = prev })
}

// inviteOnlyGate is the gate over a fake pool configured invite_only with admin group "admins".
func inviteOnlyGate(f *fakeCognito, srv *httptest.Server, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	setup := signIn{auth: &cognitoAuth{
		clientID: testClientID,
		endpoint: srv.URL + "/",
		client:   srv.Client(),
		verifier: newCognitoVerifier(f.issuer, testClientID, f.issuer+"/.well-known/jwks.json", srv.Client()),

		signUp:         signUpInviteOnly,
		adminGroup:     testAdminGroup,
		userAdminGroup: testUserAdminGroup,
		poolID:         "eu-west-2_pool",
		region:         "eu-west-2",
	}}
	registerIdentityRoutes(mux, setup, logger)
	return newIdentityGate(mux, setup, logger)
}

func adminToken(f *fakeCognito) func(*http.Request) {
	return withBearer(f.token("admin-1", map[string]any{"cognito:groups": []string{testUserAdminGroup}}))
}

func hasPolicyChars(pw string) bool {
	var lower, upper, digit bool
	for _, r := range pw {
		lower = lower || unicode.IsLower(r)
		upper = upper || unicode.IsUpper(r)
		digit = digit || unicode.IsDigit(r)
	}
	return len(pw) >= 12 && lower && upper && digit
}

func TestAuthConfigReportsTheSignUpMode(t *testing.T) {
	f, srv := newFakeCognito(t)
	// A pool built without a mode is open: an app deployed before the variable existed.
	if rec := authDo(t, gateFor(f, srv), http.MethodGet, "/api/auth/config", "", nil); rec.Code != http.StatusOK || authDecode(t, rec)["sign_up"] != "open" {
		t.Fatalf("default config = %d %s", rec.Code, rec.Body.String())
	}
	rec := authDo(t, inviteOnlyGate(f, srv, identityTestLogger()), http.MethodGet, "/api/auth/config", "", nil)
	if rec.Code != http.StatusOK || authDecode(t, rec)["sign_up"] != "invite_only" {
		t.Fatalf("invite_only config = %d %s", rec.Code, rec.Body.String())
	}
	if rec := authDo(t, composeIdentity(signIn{anonymous: true}), http.MethodGet, "/api/auth/config", "", nil); rec.Code != http.StatusOK || authDecode(t, rec)["sign_up"] != "open" {
		t.Fatalf("anonymous config = %d %s", rec.Code, rec.Body.String())
	}
}

func TestSignUpModeFromEnv(t *testing.T) {
	pool := map[string]string{"AUTH_USER_POOL_ID": "eu-west-2_abc", "AUTH_CLIENT_ID": "client", "AWS_REGION": "eu-west-2"}
	with := func(k, v string) map[string]string {
		m := map[string]string{k: v}
		for pk, pv := range pool {
			m[pk] = pv
		}
		return m
	}
	if s := signInFromEnv(identityTestEnv(pool)); s.err != nil || s.auth.signUpMode() != "open" {
		t.Fatalf("unset AUTH_SIGN_UP = %+v, want open", s)
	}
	s := signInFromEnv(identityTestEnv(with("AUTH_SIGN_UP", "invite_only")))
	if s.err != nil || s.auth.signUpMode() != "invite_only" {
		t.Fatalf("invite_only = %+v", s)
	}
	if s := signInFromEnv(identityTestEnv(with("AUTH_SIGN_UP", "closed"))); s.err == nil || s.auth != nil {
		t.Fatalf("an unknown mode = %+v, want a closed failure", s)
	}
}

func TestInviteOnlyHasNoSelfSignUpOrRecovery(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := inviteOnlyGate(f, srv, identityTestLogger())
	for _, action := range []string{"sign-up", "confirm", "resend", "forgot", "reset"} {
		rec := authDo(t, h, http.MethodPost, "/api/auth/"+action, `{"email":"a@example.invalid","password":"Long-enough-1","code":"1"}`, nil)
		if rec.Code != http.StatusNotFound || authDecode(t, rec)["code"] != "not_found" {
			t.Fatalf("POST /api/auth/%s = %d %s, want 404 not_found", action, rec.Code, rec.Body.String())
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("the pool was called: %v", f.calls)
	}

	// Signing in, and an invited person's first sign-in, still work.
	f.answer("InitiateAuth", func(map[string]any) (int, any) {
		return http.StatusOK, map[string]any{"ChallengeName": "NEW_PASSWORD_REQUIRED", "Session": "sess"}
	})
	rec := authDo(t, h, http.MethodPost, "/api/auth/sign-in", `{"email":"a@example.invalid","password":"Temp-pass-1"}`, nil)
	if rec.Code != http.StatusConflict || authDecode(t, rec)["code"] != "new_password_required" {
		t.Fatalf("invited first sign-in = %d %s", rec.Code, rec.Body.String())
	}
	f.answer("RespondToAuthChallenge", func(in map[string]any) (int, any) {
		return http.StatusOK, map[string]any{"AuthenticationResult": map[string]string{
			"IdToken": f.token("a", nil), "AccessToken": "at", "RefreshToken": "rt",
		}}
	})
	rec = authDo(t, h, http.MethodPost, "/api/auth/new-password", `{"email":"a@example.invalid","password":"My-own-pass-1","session":"sess"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("new-password = %d %s", rec.Code, rec.Body.String())
	}
}

func TestChangePasswordWorksInBothModes(t *testing.T) {
	for name, build := range map[string]func(*fakeCognito, *httptest.Server) http.Handler{
		"open": gateFor,
		"invite_only": func(f *fakeCognito, srv *httptest.Server) http.Handler {
			return inviteOnlyGate(f, srv, identityTestLogger())
		},
	} {
		t.Run(name, func(t *testing.T) {
			f, srv := newFakeCognito(t)
			h := build(f, srv)
			f.answer("InitiateAuth", func(in map[string]any) (int, any) {
				params, _ := in["AuthParameters"].(map[string]any)
				if params["USERNAME"] != "user-a@example.invalid" || params["PASSWORD"] != "Current-pass-1" {
					return http.StatusBadRequest, map[string]string{"__type": "NotAuthorizedException"}
				}
				return http.StatusOK, map[string]any{"AuthenticationResult": map[string]string{"IdToken": "x", "AccessToken": "access-token"}}
			})
			f.answer("ChangePassword", func(in map[string]any) (int, any) {
				if in["AccessToken"] != "access-token" || in["PreviousPassword"] != "Current-pass-1" || in["ProposedPassword"] != "Brand-new-pass-2" {
					return http.StatusBadRequest, map[string]string{"__type": "InvalidParameterException"}
				}
				return http.StatusOK, map[string]any{}
			})
			url := "/api/auth/change-password"
			good := `{"current_password":"Current-pass-1","new_password":"Brand-new-pass-2"}`

			if rec := authDo(t, h, http.MethodPost, url, good, nil); rec.Code != http.StatusUnauthorized {
				t.Fatalf("signed out = %d, want 401", rec.Code)
			}
			if rec := authDo(t, h, http.MethodGet, url, "", withBearer(f.token("user-a", nil))); rec.Code != http.StatusNotFound {
				t.Fatalf("GET = %d, want 404", rec.Code)
			}
			if rec := authDo(t, h, http.MethodPost, url, `{"current_password":"Wrong-pass-1","new_password":"Brand-new-pass-2"}`, withBearer(f.token("user-a", nil))); rec.Code != http.StatusUnauthorized || authDecode(t, rec)["code"] != "invalid_credentials" {
				t.Fatalf("wrong current password = %d %s", rec.Code, rec.Body.String())
			}
			if f.called("ChangePassword") {
				t.Fatal("ChangePassword was called with a wrong current password")
			}
			if rec := authDo(t, h, http.MethodPost, url, `{"current_password":"Current-pass-1"}`, withBearer(f.token("user-a", nil))); rec.Code != http.StatusBadRequest {
				t.Fatalf("no new password = %d, want 400", rec.Code)
			}
			if rec := authDo(t, h, http.MethodPost, url, good, withBearer(f.token("user-a", nil))); rec.Code != http.StatusNoContent {
				t.Fatalf("change password = %d %s, want 204", rec.Code, rec.Body.String())
			}
			f.answer("ChangePassword", func(map[string]any) (int, any) {
				return http.StatusBadRequest, map[string]string{"__type": "InvalidPasswordException", "message": "Password did not conform with policy: Password must have uppercase characters"}
			})
			rec := authDo(t, h, http.MethodPost, url, good, withBearer(f.token("user-a", nil)))
			if rec.Code != http.StatusBadRequest || authDecode(t, rec)["code"] != "invalid_password" {
				t.Fatalf("policy miss = %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestPeopleEndpointsDoNotExistInAnOpenApp(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := gateFor(f, srv)
	// Even a token in a group named "admins" gets nothing: the endpoints are not there.
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/auth/users"},
		{http.MethodPost, "/api/auth/users"},
		{http.MethodPost, "/api/auth/users/disable"},
		{http.MethodPost, "/api/auth/users/enable"},
		{http.MethodPost, "/api/auth/users/reset-password"},
	} {
		rec := authDo(t, h, c.method, c.path, `{"email":"a@example.invalid"}`, adminToken(f))
		if rec.Code != http.StatusNotFound || authDecode(t, rec)["code"] != "not_found" {
			t.Fatalf("%s %s = %d %s, want 404 not_found", c.method, c.path, rec.Code, rec.Body.String())
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("the pool was called: %v", f.calls)
	}
}

func TestPeopleEndpointsAreForTheAdminGroupOnly(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := inviteOnlyGate(f, srv, identityTestLogger())
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/auth/users"},
		{http.MethodPost, "/api/auth/users"},
		{http.MethodPost, "/api/auth/users/disable"},
		{http.MethodPost, "/api/auth/users/enable"},
		{http.MethodPost, "/api/auth/users/reset-password"},
	} {
		body := `{"email":"a@example.invalid"}`
		if rec := authDo(t, h, c.method, c.path, body, nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s signed out = %d, want 401", c.method, c.path, rec.Code)
		}
		for who, tok := range map[string]string{
			"a member":      f.token("member", nil),
			"another group": f.token("other", map[string]any{"cognito:groups": []string{"editors"}}),
			// The app's admin group (demo and verifier accounts) is not the user-admin group.
			"an app admin": f.token("app-admin", map[string]any{"cognito:groups": []string{testAdminGroup}}),
		} {
			rec := authDo(t, h, c.method, c.path, body, withBearer(tok))
			if rec.Code != http.StatusForbidden || authDecode(t, rec)["code"] != "forbidden" {
				t.Fatalf("%s %s as %s = %d %s, want 403", c.method, c.path, who, rec.Code, rec.Body.String())
			}
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("the pool was called for a non-admin: %v", f.calls)
	}
}

func TestPeopleListMarksAdminsAndSignsItsCalls(t *testing.T) {
	f, srv := newFakeCognito(t)
	h := inviteOnlyGate(f, srv, identityTestLogger())
	f.answer("ListUsers", func(in map[string]any) (int, any) {
		if in["UserPoolId"] != "eu-west-2_pool" {
			return http.StatusBadRequest, map[string]string{"__type": "InvalidParameterException"}
		}
		return http.StatusOK, map[string]any{"PaginationToken": "more", "Users": []map[string]any{
			{"Username": "ada@example.invalid", "Enabled": true, "UserStatus": "CONFIRMED", "UserCreateDate": 1.7e9,
				"Attributes": []map[string]string{{"Name": "email", "Value": "ada@example.invalid"}}},
			{"Username": "bob@example.invalid", "Enabled": false, "UserStatus": "FORCE_CHANGE_PASSWORD"},
		}}
	})
	f.answer("AdminListGroupsForUser", func(in map[string]any) (int, any) {
		if in["Username"] == "ada@example.invalid" {
			return http.StatusOK, map[string]any{"Groups": []map[string]string{{"GroupName": testUserAdminGroup}}}
		}
		return http.StatusOK, map[string]any{"Groups": []map[string]string{}}
	})
	req := httptest.NewRequest(http.MethodGet, "/api/auth/users", nil)
	adminToken(f)(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list = %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", rec.Header().Get("Cache-Control"))
	}
	var out struct {
		People []person `json:"people"`
		Next   string   `json:"next"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.People) != 2 || out.Next != "more" ||
		out.People[0].Email != "ada@example.invalid" || !out.People[0].Admin || !out.People[0].Enabled || out.People[0].CreatedAt == "" ||
		out.People[1].Admin || out.People[1].Enabled || out.People[1].Status != "FORCE_CHANGE_PASSWORD" {
		t.Fatalf("people = %+v", out)
	}
}

// signedCalls records the Authorization header of every call the fake pool receives.
func TestAdminCallsAreSigned(t *testing.T) {
	f, srv := newFakeCognito(t)
	var got []string
	inner := srv.Config.Handler
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.Header.Get("X-Amz-Target"), ".ListUsers") {
			got = append(got, r.Header.Get("Authorization")+"|"+r.Header.Get("X-Amz-Security-Token"))
		}
		inner.ServeHTTP(w, r)
	})
	h := inviteOnlyGate(f, srv, identityTestLogger())
	f.answer("ListUsers", func(map[string]any) (int, any) { return http.StatusOK, map[string]any{"Users": []any{}} })
	f.answer("AdminListGroupsForUser", func(map[string]any) (int, any) { return http.StatusOK, map[string]any{} })
	if rec := authDo(t, h, http.MethodGet, "/api/auth/users", "", adminToken(f)); rec.Code != http.StatusOK {
		t.Fatalf("list = %d", rec.Code)
	}
	if len(got) != 1 || !strings.HasPrefix(got[0], "AWS4-HMAC-SHA256 Credential=AKID/20261101/eu-west-2/cognito-idp/aws4_request") || !strings.HasSuffix(got[0], "|tok") {
		t.Fatalf("ListUsers was not SigV4-signed: %v", got)
	}
}

func TestPeopleCreateReturnsATemporaryPasswordOnceAndNeverLogsIt(t *testing.T) {
	f, srv := newFakeCognito(t)
	var logs bytes.Buffer
	h := inviteOnlyGate(f, srv, slog.New(slog.NewTextHandler(&logs, nil)))
	var sent map[string]any
	var added []string
	f.answer("AdminCreateUser", func(in map[string]any) (int, any) { sent = in; return http.StatusOK, map[string]any{} })
	f.answer("AdminAddUserToGroup", func(in map[string]any) (int, any) {
		if in["Username"] == "new.person@example.invalid" {
			added = append(added, in["GroupName"].(string))
		}
		return http.StatusOK, map[string]any{}
	})

	rec := authDo(t, h, http.MethodPost, "/api/auth/users", `{"email":"  New.Person@Example.invalid ","admin":true}`, adminToken(f))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
	}
	out := authDecode(t, rec)
	temp, _ := out["temporary_password"].(string)
	if out["email"] != "new.person@example.invalid" || out["admin"] != true || !hasPolicyChars(temp) {
		t.Fatalf("create response = %v", out)
	}
	if sent["TemporaryPassword"] != temp || sent["MessageAction"] != "SUPPRESS" || sent["Username"] != "new.person@example.invalid" || sent["UserPoolId"] != "eu-west-2_pool" {
		t.Fatalf("AdminCreateUser = %v", sent)
	}
	attrs, _ := json.Marshal(sent["UserAttributes"])
	if !strings.Contains(string(attrs), `"email_verified"`) || !strings.Contains(string(attrs), `"true"`) {
		t.Fatalf("attributes = %s, want email_verified=true", attrs)
	}
	if len(added) != 2 || added[0] != testAdminGroup || added[1] != testUserAdminGroup {
		t.Fatalf("groups added = %v, want both the admin and the user-admin group", added)
	}
	if strings.Contains(logs.String(), temp) {
		t.Fatalf("the temporary password was logged: %s", logs.String())
	}

	// Each password is its own.
	a, _ := generateTempPassword()
	b, _ := generateTempPassword()
	if a == b || !hasPolicyChars(a) || len(a) != tempLength {
		t.Fatalf("generated passwords %q %q", a, b)
	}
}

func TestPeopleCreateRefusalsAndFailuresLeakNothing(t *testing.T) {
	f, srv := newFakeCognito(t)
	var logs bytes.Buffer
	h := inviteOnlyGate(f, srv, slog.New(slog.NewTextHandler(&logs, nil)))
	for _, bad := range []string{`{"email":""}`, `{"email":"not-an-email"}`, `{"email":"a b@example.invalid"}`, `{"email":"Ada <a@example.invalid>"}`, `{"email":"a@localhost"}`} {
		if rec := authDo(t, h, http.MethodPost, "/api/auth/users", bad, adminToken(f)); rec.Code != http.StatusBadRequest {
			t.Fatalf("create %s = %d, want 400", bad, rec.Code)
		}
	}
	if f.called("AdminCreateUser") {
		t.Fatal("AdminCreateUser was called for an invalid email")
	}

	f.answer("AdminCreateUser", func(map[string]any) (int, any) {
		return http.StatusBadRequest, map[string]string{"__type": "UsernameExistsException"}
	})
	if rec := authDo(t, h, http.MethodPost, "/api/auth/users", `{"email":"a@example.invalid"}`, adminToken(f)); rec.Code != http.StatusConflict {
		t.Fatalf("existing = %d, want 409", rec.Code)
	}

	var temp string
	f.answer("AdminCreateUser", func(in map[string]any) (int, any) {
		temp, _ = in["TemporaryPassword"].(string)
		return http.StatusBadRequest, map[string]string{"__type": "AccessDeniedException", "message": "not allowed"}
	})
	rec := authDo(t, h, http.MethodPost, "/api/auth/users", `{"email":"a@example.invalid"}`, adminToken(f))
	if rec.Code != http.StatusBadGateway || strings.Contains(rec.Body.String(), temp) || strings.Contains(logs.String(), temp) {
		t.Fatalf("failure = %d %s (log %s)", rec.Code, rec.Body.String(), logs.String())
	}

	// The account exists but the group add failed: the password must still reach the admin.
	f.answer("AdminCreateUser", func(map[string]any) (int, any) { return http.StatusOK, map[string]any{} })
	f.answer("AdminAddUserToGroup", func(map[string]any) (int, any) {
		return http.StatusBadRequest, map[string]string{"__type": "ResourceNotFoundException"}
	})
	rec = authDo(t, h, http.MethodPost, "/api/auth/users", `{"email":"a@example.invalid","admin":true}`, adminToken(f))
	if rec.Code != http.StatusCreated || authDecode(t, rec)["admin"] != false || authDecode(t, rec)["temporary_password"] == "" {
		t.Fatalf("group add failure = %d %s", rec.Code, rec.Body.String())
	}
}

func TestPeopleDisableEnableAndReset(t *testing.T) {
	f, srv := newFakeCognito(t)
	var logs bytes.Buffer
	h := inviteOnlyGate(f, srv, slog.New(slog.NewTextHandler(&logs, nil)))
	calls := map[string]map[string]any{}
	for _, op := range []string{"AdminDisableUser", "AdminEnableUser", "AdminSetUserPassword"} {
		f.answer(op, func(in map[string]any) (int, any) { calls[op] = in; return http.StatusOK, map[string]any{} })
	}
	body := `{"email":"Bob@Example.invalid"}`

	rec := authDo(t, h, http.MethodPost, "/api/auth/users/disable", body, adminToken(f))
	if rec.Code != http.StatusNoContent || calls["AdminDisableUser"]["Username"] != "bob@example.invalid" {
		t.Fatalf("disable = %d %v", rec.Code, calls["AdminDisableUser"])
	}
	rec = authDo(t, h, http.MethodPost, "/api/auth/users/enable", body, adminToken(f))
	if rec.Code != http.StatusNoContent || calls["AdminEnableUser"]["Username"] != "bob@example.invalid" {
		t.Fatalf("enable = %d %v", rec.Code, calls["AdminEnableUser"])
	}

	// An administrator cannot lock themselves out.
	delete(calls, "AdminDisableUser")
	rec = authDo(t, h, http.MethodPost, "/api/auth/users/disable", `{"email":"admin-1@example.invalid"}`, adminToken(f))
	if rec.Code != http.StatusBadRequest || calls["AdminDisableUser"] != nil {
		t.Fatalf("disable self = %d, want 400 and no call", rec.Code)
	}

	rec = authDo(t, h, http.MethodPost, "/api/auth/users/reset-password", body, adminToken(f))
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("reset = %d %s (Cache-Control %q)", rec.Code, rec.Body.String(), rec.Header().Get("Cache-Control"))
	}
	temp, _ := authDecode(t, rec)["temporary_password"].(string)
	set := calls["AdminSetUserPassword"]
	if !hasPolicyChars(temp) || set["Password"] != temp || set["Permanent"] != false || set["Username"] != "bob@example.invalid" {
		t.Fatalf("AdminSetUserPassword = %v, response password %q", set, temp)
	}

	f.answer("AdminSetUserPassword", func(map[string]any) (int, any) {
		return http.StatusBadRequest, map[string]string{"__type": "UserNotFoundException"}
	})
	if rec := authDo(t, h, http.MethodPost, "/api/auth/users/reset-password", body, adminToken(f)); rec.Code != http.StatusNotFound {
		t.Fatalf("reset unknown = %d, want 404", rec.Code)
	}
	if strings.Contains(logs.String(), temp) {
		t.Fatalf("the temporary password was logged: %s", logs.String())
	}
}

func TestAdminCallsFailClosedWithoutCredentials(t *testing.T) {
	withAdminCredentials(t, awsCredentials{})
	f, srv := newFakeCognito(t)
	mux := http.NewServeMux()
	setup := signIn{auth: &cognitoAuth{
		clientID: testClientID, endpoint: srv.URL + "/", client: srv.Client(),
		verifier: newCognitoVerifier(f.issuer, testClientID, f.issuer+"/.well-known/jwks.json", srv.Client()),
		signUp:   signUpInviteOnly, adminGroup: testAdminGroup, userAdminGroup: testUserAdminGroup, poolID: "p", region: "eu-west-2",
	}}
	registerIdentityRoutes(mux, setup, identityTestLogger())
	rec := authDo(t, newIdentityGate(mux, setup, identityTestLogger()), http.MethodGet, "/api/auth/users", "", adminToken(f))
	if rec.Code != http.StatusBadGateway || f.called("ListUsers") {
		t.Fatalf("no credentials = %d, ListUsers called %v", rec.Code, f.called("ListUsers"))
	}
}

func TestPeopleFailClosedWithoutAUserAdminGroup(t *testing.T) {
	f, srv := newFakeCognito(t)
	mux := http.NewServeMux()
	setup := signIn{auth: &cognitoAuth{
		clientID: testClientID, endpoint: srv.URL + "/", client: srv.Client(),
		verifier: newCognitoVerifier(f.issuer, testClientID, f.issuer+"/.well-known/jwks.json", srv.Client()),
		signUp:   signUpInviteOnly, adminGroup: testAdminGroup, poolID: "p", region: "eu-west-2",
	}}
	registerIdentityRoutes(mux, setup, identityTestLogger())
	h := newIdentityGate(mux, setup, identityTestLogger())
	rec := authDo(t, h, http.MethodGet, "/api/auth/users", "", adminToken(f))
	if rec.Code != http.StatusForbidden || len(f.calls) != 0 {
		t.Fatalf("no user-admin group = %d, calls %v; want 403 and no pool call", rec.Code, f.calls)
	}
	if got := authDecode(t, authDo(t, h, http.MethodGet, "/api/auth/config", "", nil)); got["user_admin_group"] != nil {
		t.Fatalf("config = %v, want no user_admin_group", got)
	}
}

func TestConfigNamesTheUserAdminGroup(t *testing.T) {
	f, srv := newFakeCognito(t)
	rec := authDo(t, inviteOnlyGate(f, srv, identityTestLogger()), http.MethodGet, "/api/auth/config", "", nil)
	if got := authDecode(t, rec)["user_admin_group"]; got != testUserAdminGroup {
		t.Fatalf("config user_admin_group = %v", got)
	}
}
