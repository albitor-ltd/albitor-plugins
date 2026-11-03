package shell

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"webapp/internal/apikit"
	"webapp/internal/identity"
)

// The sign-in routes under /api/auth/: the authorization-code flow with PKCE against Cognito's
// hosted sign-in page, sign-out and account deletion. The browser is sent to GET /api/auth/login,
// which redirects to Cognito's hosted sign-in; Cognito sends it back to GET /api/auth/callback
// with a one-time code; and the API trades the code for tokens itself.
//
// registerAuthRoutes is called from registerIdentityRoutes in identity_verify.go.

// registerAuthRoutes answers /api/auth/*: the sign-in routes, and in a local run, which has no
// pool, a page that says why.
func registerAuthRoutes(mux *http.ServeMux, setup signIn, logger *slog.Logger) {
	mux.HandleFunc(authPathRoot, func(w http.ResponseWriter, r *http.Request) {
		// Sign-out is the one call a plain HTML form makes: the "Sign out" button is a POST form,
		// never a link, so a link prefetch or another site's link cannot sign anyone out. Like a
		// SvelteKit form action, a form-encoded POST is refused unless the browser says it came
		// from this site. It is handled here, before the local-run answers, so the button works
		// there too.
		if r.Method == http.MethodPost && r.URL.Path == authPathRoot+"sign-out" && isFormRequest(r) {
			if !sameOrigin(r) {
				writeAuthError(w, http.StatusForbidden, "forbidden", "cross-site form post forbidden")
				return
			}
			if setup.auth != nil && !setup.auth.local {
				setup.auth.revokeAndClear(w, r)
				http.Redirect(w, r, setup.auth.logoutURL, http.StatusSeeOther)
				return
			}
			// A local run has no pool, so no Cognito session to end.
			clearSessionCookies(w)
			http.Redirect(w, r, signInPath, http.StatusSeeOther)
			return
		}
		if setup.auth == nil {
			if r.Method == http.MethodGet {
				// A browser that followed the sign-in link lands on a page that says why, not on JSON.
				http.Redirect(w, r, signInErrorPath("local"), http.StatusFound)
				return
			}
			apikit.WriteJSON(w, http.StatusNotImplemented, map[string]string{
				"error": "sign-in is not available in a local run", "code": "local",
			})
			return
		}
		setup.auth.serveAuth(w, r, logger)
	})
}

const (
	maxAuthBodyBytes = 16 << 10
	maxNextBytes     = 2048

	// adminScope is what Cognito's DeleteUser requires of the access token. The pool's client
	// must be granted it (the hosted module's scopes) for self-service deletion to work.
	adminScope = "aws.cognito.signin.user.admin"
)

// The web routes the API sends the browser to.
const (
	signInPath  = "/sign-in"
	refusedPath = "/not-allowed"
)

// The transient cookies of one sign-in, set by /api/auth/login and read by /api/auth/callback.
// They are SameSite=Lax, not Strict, on purpose: the callback is a top-level navigation from
// Cognito's site, and a Strict cookie is not sent on it. They last minutes.
const (
	stateCookie    = "__Host-auth_state"
	verifierCookie = "__Host-auth_verifier"
	nextCookie     = "__Host-auth_next"
	// transientMaxAge bounds a sign-in: the person has this long on Cognito's page.
	transientMaxAge = 10 * time.Minute
)

// OnAccountDeleted, when an app sets it, is called after a person's account has been deleted, so
// the app can remove what it holds for them (rows, files). It runs once, with the identity the
// session had. The account is already gone, so a failure is logged and not retried by the
// person: make the hook idempotent, and make an erasure job the backstop. Set it from the app's
// own code (an init in a file of its own), never by editing this one.
var OnAccountDeleted func(ctx context.Context, id identity.Identity) error

func setTransientCookie(w http.ResponseWriter, name, value string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     cookiePath,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearTransientCookies(w http.ResponseWriter) {
	for _, name := range []string{stateCookie, verifierCookie, nextCookie} {
		setTransientCookie(w, name, "", -time.Second)
	}
}

// serveAuth answers /api/auth/<action>: login and callback (GET), sign-out and delete (POST).
func (a *hostedAuth) serveAuth(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	action := strings.TrimPrefix(r.URL.Path, authPathRoot)
	switch {
	case r.Method == http.MethodGet && action == "login":
		a.login(w, r)
	case r.Method == http.MethodGet && action == "callback":
		a.callback(w, r, logger)
	case r.Method == http.MethodPost && (action == "sign-out" || action == "delete"):
		if a.local {
			// A local run has no pool to sign out of. A spec signs in through the local issuer
			// instead (web/tests/helpers/local-auth.ts).
			writeAuthError(w, http.StatusNotImplemented, "local", "sign-in is not available in a local run: use the local issuer")
			return
		}
		// JSON only. Another site's HTML form cannot send this content type without the browser
		// asking first, so it cannot sign a visitor out or delete their account.
		if mt, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";"); !strings.EqualFold(strings.TrimSpace(mt), "application/json") {
			writeAuthError(w, http.StatusUnsupportedMediaType, "invalid_input", "send application/json")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)
		if action == "sign-out" {
			a.signOut(w, r)
		} else {
			a.deleteAccount(w, r, logger)
		}
	default:
		apikit.WriteError(w, http.StatusNotFound, "not found")
	}
}

// signInErrorPath is where the browser goes when a sign-in cannot finish. The page shows the
// reason and does not redirect again, so a failure cannot loop.
func signInErrorPath(code string) string { return signInPath + "?error=" + url.QueryEscape(code) }

// safeNext returns next when it is a path on this site the person can be sent to after signing
// in, and "/" otherwise. A scheme, a host, a protocol-relative "//", a backslash (some browsers
// read it as a slash), a control character or a path the API or the sign-in screens own is not.
func safeNext(next string) string {
	if next == "" || len(next) > maxNextBytes || next[0] != '/' {
		return "/"
	}
	if strings.HasPrefix(next, "//") || strings.ContainsAny(next, "\\") {
		return "/"
	}
	for _, r := range next {
		if r < 0x20 || r == 0x7f {
			return "/"
		}
	}
	u, err := url.Parse(next)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || !strings.HasPrefix(u.Path, "/") {
		return "/"
	}
	switch p := u.Path; {
	case p == "/api" || strings.HasPrefix(p, "/api/"):
		return "/"
	case p == signInPath || p == "/sign-out" || p == refusedPath:
		return "/"
	}
	return next
}

// login is GET /api/auth/login?next=: it starts a sign-in. It makes a random state and a PKCE
// verifier, keeps both (and where to go afterwards) in short-lived cookies, and sends the browser
// to the hosted sign-in page with the state and the verifier's S256 challenge.
func (a *hostedAuth) login(w http.ResponseWriter, r *http.Request) {
	if a.local {
		http.Redirect(w, r, signInErrorPath("local"), http.StatusFound)
		return
	}
	state, err := randomToken(32)
	if err != nil {
		writeAuthError(w, http.StatusInternalServerError, "unavailable", "sign-in is unavailable: try again shortly")
		return
	}
	verifier, err := randomToken(32)
	if err != nil {
		writeAuthError(w, http.StatusInternalServerError, "unavailable", "sign-in is unavailable: try again shortly")
		return
	}
	challenge := pkceChallenge(verifier)
	next := safeNext(r.URL.Query().Get("next"))

	setTransientCookie(w, stateCookie, state, transientMaxAge)
	// The issue time travels with the verifier, so the API can refuse a stale one whatever a
	// browser does with max-age. A verifier is base64url, which has no ".".
	setTransientCookie(w, verifierCookie, verifier+"."+strconv.FormatInt(a.now().Unix(), 10), transientMaxAge)
	setTransientCookie(w, nextCookie, base64.RawURLEncoding.EncodeToString([]byte(next)), transientMaxAge)

	// The login URL is appended to, not rebuilt, so the scope and redirect_uri stay exactly as
	// the deploy wrote them (the scope is the deploy's to grant, never the app's).
	to := a.loginURL + "&state=" + state + "&code_challenge=" + challenge + "&code_challenge_method=S256"
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, to, http.StatusFound)
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// callback is GET /api/auth/callback: it finishes a sign-in. The order is the point. The state is
// checked first, before the code goes anywhere; the ID token is verified like any other; the
// address must be verified; and only then does the domain rule run, so a refused address never
// holds a session.
func (a *hostedAuth) callback(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	w.Header().Set("Cache-Control", "no-store")
	if a.local {
		http.Redirect(w, r, signInErrorPath("local"), http.StatusFound)
		return
	}
	fail := func(code, why string, attrs ...any) {
		clearTransientCookies(w)
		logger.WarnContext(r.Context(), "sign-in did not finish",
			append([]any{slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())), slog.String("reason", why)}, attrs...)...)
		http.Redirect(w, r, signInErrorPath(code), http.StatusFound)
	}

	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		// The person declined, or Cognito could not sign them in. There is nothing to exchange.
		fail("denied", "the hosted sign-in answered an error", slog.String("error", e))
		return
	}
	stateC, errS := r.Cookie(stateCookie)
	verifierC, errV := r.Cookie(verifierCookie)
	if errS != nil || errV != nil || stateC.Value == "" || verifierC.Value == "" {
		fail("expired", "the sign-in cookies are missing")
		return
	}
	verifier, issued, ok := strings.Cut(verifierC.Value, ".")
	issuedAt, perr := strconv.ParseInt(issued, 10, 64)
	age := a.now().Sub(time.Unix(issuedAt, 0))
	if !ok || verifier == "" || perr != nil || age > transientMaxAge || age < -time.Minute {
		fail("expired", "the sign-in verifier is malformed or too old")
		return
	}
	state := q.Get("state")
	if state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(stateC.Value)) != 1 {
		fail("failed", "the state does not match the one this browser started with")
		return
	}
	code := q.Get("code")
	if code == "" || len(code) > 2048 {
		fail("failed", "no authorization code")
		return
	}
	next := "/"
	if c, err := r.Cookie(nextCookie); err == nil {
		if b, derr := base64.RawURLEncoding.DecodeString(c.Value); derr == nil {
			next = safeNext(string(b))
		}
	}

	tokens, terr := a.tokenRequest(r.Context(), url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {a.clientID},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {a.redirectURI},
	})
	if terr != nil {
		if terr.unreachable() {
			fail("unavailable", "the token endpoint could not be reached", slog.String("error", terr.Error()))
		} else {
			fail("failed", "the code was refused", slog.String("error", terr.Error()))
		}
		return
	}
	claims, err := a.verifier.verifyClaims(r.Context(), tokens.IDToken)
	if errors.Is(err, errVerifierUnavailable) {
		fail("unavailable", "the pool's keys could not be fetched", slog.String("error", err.Error()))
		return
	}
	if err == nil {
		err = a.acceptIDToken(claims)
	}
	switch {
	case err == nil:
	case errors.Is(err, errEmailUnverified):
		fail("unverified", "the email address is missing or not verified")
		return
	case errors.Is(err, errEmailNotAllowed):
		clearTransientCookies(w)
		logger.InfoContext(r.Context(), "sign-in refused: email domain not allowed",
			slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())))
		http.Redirect(w, r, refusedPath, http.StatusFound)
		return
	default:
		fail("failed", "the pool issued a token this API does not accept", slog.Any("error", err))
		return
	}
	if !cookieFits(tokens.IDToken, tokens.RefreshToken) {
		fail("toolarge", "the tokens are too large to keep in cookies")
		return
	}

	setSessionCookie(w, sessionCookie, tokens.IDToken, sessionMaxAge)
	if tokens.RefreshToken != "" {
		setSessionCookie(w, refreshCookie, tokens.RefreshToken, sessionMaxAge)
	}
	clearTransientCookies(w)
	writeNavigation(w, next)
}

// writeNavigation sends the browser on to a page of this site WITHOUT a redirect.
//
// The session cookies are SameSite=Strict. The callback is reached by a cross-site top-level
// navigation (from Cognito), and a browser does not send Strict cookies on a request whose
// navigation chain began on another site, which includes the redirect a 302 would make from the
// callback to the page: that first request would arrive with no session, and the guard would send
// the person straight back to sign in. The cookies ARE stored from this response. So the answer is
// a 200 page that navigates itself (a meta refresh): a new navigation started by a document of this
// site is same-site, and carries them.
func writeNavigation(w http.ResponseWriter, next string) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	target := html.EscapeString(next)
	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta http-equiv="refresh" content="0;url=%s">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Signing you in</title>
</head>
<body>
<p>Signing you in. <a href="%s">Continue</a> if nothing happens.</p>
</body>
</html>
`, target, target)
}

// signOut is the JSON form of sign-out: it answers the hosted sign-out URL, which the caller
// navigates to so Cognito's own cookie is cleared too. Without that, the next sign-in would skip
// the password. The Sign out button uses the form post instead, which answers 303 to that URL.
func (a *hostedAuth) signOut(w http.ResponseWriter, r *http.Request) {
	a.revokeAndClear(w, r)
	apikit.WriteJSON(w, http.StatusOK, map[string]string{"logout_url": a.logoutURL})
}

// revokeAndClear revokes the refresh token, best effort, and clears the session: the work of
// sign-out, for both its JSON and its form-post forms.
func (a *hostedAuth) revokeAndClear(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookie); err == nil && c.Value != "" {
		_ = a.revoke(r.Context(), c.Value)
	}
	clearSessionCookies(w)
}

// isFormRequest reports whether the body is one an HTML form can send without a preflight.
func isFormRequest(r *http.Request) bool {
	mt, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
	switch strings.ToLower(strings.TrimSpace(mt)) {
	case "application/x-www-form-urlencoded", "multipart/form-data", "text/plain":
		return true
	}
	return false
}

// sameOrigin is SvelteKit's form-action CSRF rule: the browser's Sec-Fetch-Site says same-origin,
// or its Origin header names this host (X-Forwarded-Host first, as the app sits behind a CDN).
// A form post with neither is refused.
func sameOrigin(r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "same-origin" {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return strings.EqualFold(u.Host, host)
}

// revoke asks the token endpoint to revoke a refresh token.
func (a *hostedAuth) revoke(ctx context.Context, token string) error {
	ctx, cancel := context.WithTimeout(ctx, hostedTimeout)
	defer cancel()
	form := url.Values{"token": {token}, "client_id": {a.clientID}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.revokeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxTokenBytes))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("revoke answered %d", resp.StatusCode)
	}
	return nil
}

// deleteAccount is POST /api/auth/delete: the signed-in person deletes their own account.
//
// Cognito's DeleteUser takes the person's ACCESS token and needs the admin scope on it. The API
// holds no access token (the session is the ID token), so it trades the refresh cookie for a fresh
// one first, which also means a stale session cannot delete an account. A token without the scope
// is refused here with a clear code, and Cognito's own refusal maps to the same.
func (a *hostedAuth) deleteAccount(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	id, ok := a.authenticate(w, r, logger)
	if !ok {
		return
	}
	reauth := func() {
		writeAuthError(w, http.StatusUnauthorized, "session_expired", "sign in again to delete your account")
	}
	c, err := r.Cookie(refreshCookie)
	if err != nil || c.Value == "" {
		reauth()
		return
	}
	tokens, terr := a.tokenRequest(r.Context(), url.Values{
		"grant_type": {"refresh_token"}, "client_id": {a.clientID}, "refresh_token": {c.Value},
	})
	if terr != nil {
		if terr.unreachable() {
			writeAuthError(w, http.StatusBadGateway, "unavailable", "sign-in is unavailable: try again shortly")
		} else {
			clearSessionCookies(w)
			reauth()
		}
		return
	}
	access, err := a.verifier.verifyClaims(r.Context(), tokens.AccessToken)
	if err != nil || access.TokenUse != "access" || access.Sub != id.Sub {
		if errors.Is(err, errVerifierUnavailable) {
			writeAuthError(w, http.StatusBadGateway, "unavailable", "sign-in is unavailable: try again shortly")
			return
		}
		reauth()
		return
	}
	if !access.hasScope(adminScope) {
		writeAuthError(w, http.StatusForbidden, "scope_missing", "this app cannot delete accounts yet: its sign-in is not set up to allow it")
		return
	}
	if cerr := a.deleteUser(r.Context(), tokens.AccessToken); cerr != nil {
		switch {
		case cerr.kind == "NotAuthorizedException" && strings.Contains(strings.ToLower(cerr.message), "scope"):
			writeAuthError(w, http.StatusForbidden, "scope_missing", "this app cannot delete accounts yet: its sign-in is not set up to allow it")
		case cerr.kind == "NotAuthorizedException" || cerr.kind == "UserNotFoundException":
			clearSessionCookies(w)
			reauth()
		case cerr.kind == "":
			logger.ErrorContext(r.Context(), "account deletion could not reach the pool",
				slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())),
				slog.String("error", cerr.Error()))
			writeAuthError(w, http.StatusBadGateway, "unavailable", "sign-in is unavailable: try again shortly")
		default:
			logger.ErrorContext(r.Context(), "account deletion failed",
				slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())),
				slog.String("error", cerr.Error()))
			writeAuthError(w, http.StatusBadGateway, "unavailable", "your account could not be deleted: try again shortly")
		}
		return
	}
	clearSessionCookies(w)
	if hook := OnAccountDeleted; hook != nil {
		if err := hook(r.Context(), id); err != nil {
			logger.ErrorContext(r.Context(), "the account was deleted but the app's data for it was not removed",
				slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())),
				slog.String("sub", id.Sub),
				slog.String("error", err.Error()))
		}
	}
	apikit.WriteJSON(w, http.StatusOK, map[string]string{"logout_url": a.logoutURL})
}

// cognitoError is a failed user-pool call. kind is Cognito's exception name, or empty when the
// call never got an answer from Cognito.
type cognitoError struct {
	kind    string
	message string
	err     error
}

func (e *cognitoError) Error() string {
	if e.kind == "" {
		return fmt.Sprintf("cognito unreachable: %v", e.err)
	}
	return e.kind + ": " + e.message
}

func (e *cognitoError) Unwrap() error { return e.err }

// deleteUser calls the user pool's DeleteUser with the person's access token. It needs no AWS
// credentials: the access token is the whole of the authority.
func (a *hostedAuth) deleteUser(ctx context.Context, accessToken string) *cognitoError {
	payload, err := json.Marshal(map[string]string{"AccessToken": accessToken})
	if err != nil {
		return &cognitoError{err: err}
	}
	ctx, cancel := context.WithTimeout(ctx, hostedTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(payload))
	if err != nil {
		return &cognitoError{err: err}
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService.DeleteUser")
	resp, err := a.client.Do(req)
	if err != nil {
		return &cognitoError{err: err}
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTokenBytes))
	if err != nil {
		return &cognitoError{err: err}
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	var e struct {
		Type    string `json:"__type"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &e)
	kind := e.Type
	if i := strings.LastIndex(kind, "#"); i >= 0 {
		kind = kind[i+1:]
	}
	if kind == "" {
		return &cognitoError{err: fmt.Errorf("DeleteUser answered %d", resp.StatusCode)}
	}
	return &cognitoError{kind: kind, message: e.Message}
}

func writeAuthError(w http.ResponseWriter, status int, code, msg string) {
	apikit.WriteJSON(w, status, map[string]string{"error": msg, "code": code})
}
