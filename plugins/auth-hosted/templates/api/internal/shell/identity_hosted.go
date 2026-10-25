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
	"regexp"
	"strconv"
	"strings"
	"time"

	"webapp/internal/apikit"
	"webapp/internal/identity"
	"webapp/internal/localauth"
)

// The hosted half of sign-in: where the user pool comes from, the authorization-code flow with
// PKCE against Cognito's hosted sign-in page, and the session cookies.
//
// WHY THE API RUNS THE FLOW. The page is a static build that carries no pool ids, and its
// Content-Security-Policy allows connections to its own origin only. So the browser is sent to
// GET /api/auth/login, which redirects to Cognito's hosted sign-in; Cognito sends it back to
// GET /api/auth/callback with a one-time code; and the API trades the code for tokens itself. The
// tokens become HttpOnly cookies the page's JavaScript cannot read, so there is no token in
// localStorage for an injected script to steal, and the app never sees a password.

// The environment the hosted deploy hands the function (the auth module's `environment` output),
// and the one optional rule the platform adds.
const (
	envIssuer        = "AUTH_ISSUER"
	envClientID      = "AUTH_CLIENT_ID"
	envJWKSURL       = "AUTH_JWKS_URL"
	envPosture       = "AUTH_POSTURE"
	envLoginURL      = "AUTH_LOGIN_URL"
	envLogoutURL     = "AUTH_LOGOUT_URL"
	envSignupDomains = "SIGNUP_DOMAINS"
	envLambdaName    = "AWS_LAMBDA_FUNCTION_NAME"
	envLocalRun      = "ALBITOR_LOCAL"
)

// The session cookies. Both HttpOnly, Secure and SameSite=Strict: no script reads them and no
// other site's request sends them.
const (
	//
	// The __Host- prefix makes a browser refuse the cookie unless it is Secure, has Path=/ and no
	// Domain, and it stops a sibling subdomain (every app is a subdomain of one shared domain) from
	// planting a cookie of the same name that this site would then read. The cost is that the
	// refresh cookie is sent with every request to this site, not only the API's.
	sessionCookie = "__Host-session"
	refreshCookie = "__Host-refresh"
	cookiePath    = "/"
	// maxCookieValue keeps a cookie under the ~4096 bytes a browser stores. One that does not fit is
	// dropped silently, which would loop the person through sign-in, so it is refused instead.
	maxCookieValue = 4000
	// sessionMaxAge matches the public client's refresh-token validity.
	sessionMaxAge = 30 * 24 * time.Hour
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

const (
	maxAuthBodyBytes = 16 << 10
	maxTokenBytes    = 1 << 20
	hostedTimeout    = 10 * time.Second
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

// signIn is how this process answers who a request is, decided once from the environment: a pool,
// the local issuer, nobody (a local run with no sign-in), or a failure, which closes every route.
type signIn struct {
	auth      *hostedAuth
	anonymous bool
	err       error
}

// signInFromEnv decides where identity comes from:
//
//   - any of AUTH_ISSUER, AUTH_CLIENT_ID, AUTH_JWKS_URL, AUTH_POSTURE, AUTH_LOGIN_URL and
//     AUTH_LOGOUT_URL set: that pool, and all six must be set and well formed. A partial
//     set is a failure, never a fallback.
//   - a deployed function (AWS_LAMBDA_FUNCTION_NAME) with none of them: a failure.
//   - a local run (ALBITOR_LOCAL=1) with ALBITOR_LOCAL_AUTH=1: the local issuer api-go starts,
//     alone. The sign-in routes say they are not available.
//   - a local run with neither: nobody, and every request acts as the anonymous owner.
//   - anything else: a failure. An API that is neither local nor configured refuses everything
//     rather than guess.
//
// The pool and the deployed marker are decided before the local issuer, so a named pool or a
// deployed function never trusts it, whatever else is set.
func signInFromEnv(getenv func(string) string) signIn {
	configured := false
	for _, name := range []string{envIssuer, envClientID, envJWKSURL, envPosture, envLoginURL, envLogoutURL} {
		if strings.TrimSpace(getenv(name)) != "" {
			configured = true
		}
	}
	if configured {
		cfg, err := hostedConfigFromEnv(getenv)
		if err != nil {
			return signIn{err: err}
		}
		return signIn{auth: newHostedAuth(cfg)}
	}
	if strings.TrimSpace(getenv(envLambdaName)) != "" {
		return signIn{err: fmt.Errorf("a deployed function has no user pool: %s and the other AUTH_* variables are not set", envIssuer)}
	}
	if strings.TrimSpace(getenv(envLocalRun)) != "1" {
		return signIn{err: errors.New("no user pool is configured and this is not a local run (ALBITOR_LOCAL=1)")}
	}
	if strings.TrimSpace(getenv(localauth.EnabledEnv)) == "1" {
		return signIn{auth: newLocalIssuerAuth(localauth.IssuerURL(getenv("PORT")))}
	}
	return signIn{anonymous: true}
}

// hostedConfig is the pool, the client and the hosted pages the API trusts.
type hostedConfig struct {
	Issuer        string
	ClientID      string
	JWKSURL       string
	Posture       string
	LoginURL      string
	LogoutURL     string
	SignupDomains []string
}

var domainEntry = regexp.MustCompile(`^(\*\.)?[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

// hostedConfigFromEnv reads and checks the whole contract. Every URL must be https, because the
// API sends people to them and posts codes to them.
func hostedConfigFromEnv(getenv func(string) string) (hostedConfig, error) {
	get := func(name string) string { return strings.TrimSpace(getenv(name)) }
	cfg := hostedConfig{
		Issuer:    get(envIssuer),
		ClientID:  get(envClientID),
		JWKSURL:   get(envJWKSURL),
		Posture:   get(envPosture),
		LoginURL:  get(envLoginURL),
		LogoutURL: get(envLogoutURL),
	}
	for name, v := range map[string]string{
		envIssuer: cfg.Issuer, envClientID: cfg.ClientID, envJWKSURL: cfg.JWKSURL,
		envPosture: cfg.Posture, envLoginURL: cfg.LoginURL, envLogoutURL: cfg.LogoutURL,
	} {
		if v == "" {
			return hostedConfig{}, fmt.Errorf("%s is not set: all of the AUTH_* variables must be set", name)
		}
	}
	if !apikit.IsSafeToken(cfg.ClientID, 128) {
		return hostedConfig{}, fmt.Errorf("%s is not well formed", envClientID)
	}
	// "public" is accepted so a deploy that sets it still starts, and is treated exactly as "private":
	// every route needs a person. Per-route public endpoints are a later feature; a deployed function
	// never acts for the anonymous owner.
	if cfg.Posture != "private" && cfg.Posture != "public" {
		return hostedConfig{}, fmt.Errorf("%s must be private or public", envPosture)
	}
	if _, err := httpsURL(envIssuer, cfg.Issuer); err != nil {
		return hostedConfig{}, err
	}
	if _, err := httpsURL(envJWKSURL, cfg.JWKSURL); err != nil {
		return hostedConfig{}, err
	}
	login, err := httpsURL(envLoginURL, cfg.LoginURL)
	if err != nil {
		return hostedConfig{}, err
	}
	q := login.Query()
	if login.Path != "/oauth2/authorize" || q.Get("client_id") != cfg.ClientID || q.Get("response_type") != "code" {
		return hostedConfig{}, fmt.Errorf("%s is not this client's /oauth2/authorize request for a code", envLoginURL)
	}
	for _, p := range []string{"state", "code_challenge", "code_challenge_method"} {
		if q.Has(p) {
			return hostedConfig{}, fmt.Errorf("%s already carries %s, which the API adds", envLoginURL, p)
		}
	}
	redirect, err := httpsURL("redirect_uri in "+envLoginURL, q.Get("redirect_uri"))
	if err != nil {
		return hostedConfig{}, err
	}
	if redirect.Path != "/api/auth/callback" {
		return hostedConfig{}, fmt.Errorf("redirect_uri in %s must be this API's /api/auth/callback", envLoginURL)
	}
	logout, err := httpsURL(envLogoutURL, cfg.LogoutURL)
	if err != nil {
		return hostedConfig{}, err
	}
	if logout.Path != "/logout" || logout.Query().Get("client_id") != cfg.ClientID {
		return hostedConfig{}, fmt.Errorf("%s is not this client's /logout request", envLogoutURL)
	}
	domains, err := parseSignupDomains(getenv(envSignupDomains))
	if err != nil {
		return hostedConfig{}, err
	}
	cfg.SignupDomains = domains
	return cfg, nil
}

func httpsURL(name, raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return nil, fmt.Errorf("%s must be an https URL", name)
	}
	return u, nil
}

// parseSignupDomains reads SIGNUP_DOMAINS: email domains separated by commas or spaces, each
// `example.org` (exactly that domain) or `*.example.org` (any subdomain of it). A leading `@`
// is allowed. Empty means anyone may sign in. An entry that is neither form is an error, so a
// typo closes the API rather than quietly opening or emptying the rule.
func parseSignupDomains(raw string) ([]string, error) {
	var out []string
	for _, entry := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' }) {
		entry = strings.TrimPrefix(strings.ToLower(entry), "@")
		if !domainEntry.MatchString(entry) {
			return nil, fmt.Errorf("%s holds %q, which is not a domain such as example.org or *.example.org", envSignupDomains, entry)
		}
		out = append(out, entry)
	}
	return out, nil
}

// signupAllowed reports whether an email address's domain is permitted. With no rule, it is.
func signupAllowed(domains []string, email string) bool {
	if len(domains) == 0 {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || at == len(email)-1 {
		return false
	}
	domain := strings.ToLower(email[at+1:])
	for _, d := range domains {
		if suffix, ok := strings.CutPrefix(d, "*"); ok {
			if strings.HasSuffix(domain, suffix) && len(domain) > len(suffix) {
				return true
			}
			continue
		}
		if domain == d {
			return true
		}
	}
	return false
}

// OnAccountDeleted, when an app sets it, is called after a person's account has been deleted, so
// the app can remove what it holds for them (rows, files). It runs once, with the identity the
// session had. The account is already gone, so a failure is logged and not retried by the
// person: make the hook idempotent, and make an erasure job the backstop. Set it from the app's
// own code (an init in a file of its own), never by editing this one.
var OnAccountDeleted func(ctx context.Context, id identity.Identity) error

// hostedAuth is one pool and client: the token check and the hosted-sign-in flow. With local set
// it is api-go's local issuer instead: tokens are checked the same way, and there is no pool to
// sign in to.
type hostedAuth struct {
	clientID    string
	loginURL    string // as given, less the per-request parameters
	logoutURL   string
	redirectURI string
	tokenURL    string // the hosted domain's /oauth2/token
	revokeURL   string // the hosted domain's /oauth2/revoke
	endpoint    string // the user pool's API, for DeleteUser
	domains     []string
	client      *http.Client
	verifier    *cognitoVerifier
	local       bool
	now         func() time.Time
}

// newLocalIssuerAuth trusts the local issuer at issuer, and nothing else.
func newLocalIssuerAuth(issuer string) *hostedAuth {
	client := &http.Client{Timeout: hostedTimeout}
	return &hostedAuth{
		clientID: localauth.ClientID,
		client:   client,
		verifier: newCognitoVerifier(issuer, localauth.ClientID, issuer+"/.well-known/jwks.json", client),
		local:    true,
		now:      time.Now,
	}
}

// newHostedAuth builds from a checked config.
func newHostedAuth(cfg hostedConfig) *hostedAuth {
	client := &http.Client{
		Timeout: hostedTimeout,
		// A token endpoint answers; it never redirects. Following one would send the code elsewhere.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	login, _ := url.Parse(cfg.LoginURL)
	issuer, _ := url.Parse(cfg.Issuer)
	host := login.Scheme + "://" + login.Host
	return &hostedAuth{
		clientID:    cfg.ClientID,
		loginURL:    cfg.LoginURL,
		logoutURL:   cfg.LogoutURL,
		redirectURI: login.Query().Get("redirect_uri"),
		tokenURL:    host + "/oauth2/token",
		revokeURL:   host + "/oauth2/revoke",
		endpoint:    issuer.Scheme + "://" + issuer.Host + "/",
		domains:     cfg.SignupDomains,
		client:      client,
		verifier:    newCognitoVerifier(cfg.Issuer, cfg.ClientID, cfg.JWKSURL, client),
		now:         time.Now,
	}
}

// authenticate returns the verified identity a request carries: from its bearer token, or from the
// session cookie, refreshing an expired session from the refresh cookie. On a refusal it has
// already written the response.
//
// EVERY token that gets a request in is held to acceptIDToken, bearer or cookie, because the
// client is public: anyone can run the authorization code flow themselves and present the tokens.
// So the gate takes ID tokens only (an access token carries no email to check), with a verified
// address the domain rule allows.
func (a *hostedAuth) authenticate(w http.ResponseWriter, r *http.Request, logger *slog.Logger) (identity.Identity, bool) {
	raw, fromCookie := requestToken(r)
	if raw == "" {
		writeUnauthenticated(w)
		return identity.Identity{}, false
	}
	claims, err := a.verifier.verifyClaims(r.Context(), raw)
	if err == nil {
		err = a.acceptIDToken(claims)
	}
	if errors.Is(err, errTokenExpired) && fromCookie {
		claims, err = a.refreshSession(w, r)
	}
	switch {
	case err == nil:
		return identity.Identity{Sub: claims.Sub, Email: claims.Email, Groups: claims.Groups}, true
	case errors.Is(err, errVerifierUnavailable):
		logger.ErrorContext(r.Context(), "could not check a sign-in token",
			slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())),
			slog.String("error", err.Error()))
		apikit.WriteError(w, http.StatusServiceUnavailable, "sign-in is unavailable")
		return identity.Identity{}, false
	case errors.Is(err, errEmailNotAllowed):
		if fromCookie {
			clearSessionCookies(w)
		}
		apikit.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "that email address cannot use this service", "code": "email_not_allowed",
		})
		return identity.Identity{}, false
	default:
		// An unknown kid may only mean the pool rotated its keys: refuse the request but keep the
		// cookies, so the person is not signed out (and their refresh token lost) by a rotation.
		if fromCookie && !errors.Is(err, errUnknownKey) {
			clearSessionCookies(w)
		}
		writeUnauthenticated(w)
		return identity.Identity{}, false
	}
}

var (
	// errEmailNotAllowed is an ID token whose address the sign-up domain rule refuses.
	errEmailNotAllowed = errors.New("email domain is not allowed")
	// errEmailUnverified is an ID token whose address is missing or not verified.
	errEmailUnverified = errors.New("email address is not verified")
)

// acceptIDToken is the one rule for who may hold a session, applied to every ID token the API
// takes: at the callback, on a refresh, and on each request. It must be an ID token, for an address
// that Cognito has verified, which SIGNUP_DOMAINS allows. A person can change their own address to
// another verified one, so it is the token's address, every time (a string comparison, no call).
func (a *hostedAuth) acceptIDToken(c jwtClaims) error {
	if c.TokenUse != "id" {
		return errTokenInvalid
	}
	email := strings.ToLower(strings.TrimSpace(c.Email))
	if email == "" || !bool(c.EmailVerified) {
		return errEmailUnverified
	}
	if !signupAllowed(a.domains, email) {
		return errEmailNotAllowed
	}
	return nil
}

func writeUnauthenticated(w http.ResponseWriter) {
	apikit.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "sign-in required", "code": "unauthenticated"})
}

// requestToken returns the request's token and whether it came from the session cookie. A bearer
// header wins.
func requestToken(r *http.Request) (string, bool) {
	if h := r.Header.Get("Authorization"); h != "" {
		if token, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(token), false
		}
		return "", false
	}
	if c, err := r.Cookie(sessionCookie); err == nil {
		return c.Value, true
	}
	return "", false
}

// refreshSession swaps the refresh cookie for a new session, at the token endpoint. The new ID
// token is held to acceptIDToken like any other, so a refusal ends the session.
func (a *hostedAuth) refreshSession(w http.ResponseWriter, r *http.Request) (jwtClaims, error) {
	c, err := r.Cookie(refreshCookie)
	if a.local || err != nil || c.Value == "" {
		return jwtClaims{}, errTokenInvalid
	}
	tokens, terr := a.tokenRequest(r.Context(), url.Values{
		"grant_type": {"refresh_token"}, "client_id": {a.clientID}, "refresh_token": {c.Value},
	})
	if terr != nil {
		if terr.unreachable() {
			return jwtClaims{}, fmt.Errorf("%w: %w", errVerifierUnavailable, terr)
		}
		return jwtClaims{}, errTokenInvalid
	}
	claims, err := a.verifier.verifyClaims(r.Context(), tokens.IDToken)
	if err != nil {
		return jwtClaims{}, err
	}
	if err := a.acceptIDToken(claims); err != nil {
		return jwtClaims{}, err
	}
	if !cookieFits(tokens.IDToken, tokens.RefreshToken) {
		return jwtClaims{}, errTokenInvalid
	}
	setSessionCookie(w, sessionCookie, tokens.IDToken, sessionMaxAge)
	// A pool that rotates refresh tokens hands back a new one: keep it, or the next refresh fails.
	if tokens.RefreshToken != "" {
		setSessionCookie(w, refreshCookie, tokens.RefreshToken, sessionMaxAge)
	}
	return claims, nil
}

// cookieFits reports whether every value is small enough to be stored as a cookie.
func cookieFits(values ...string) bool {
	for _, v := range values {
		if len(v) > maxCookieValue {
			return false
		}
	}
	return true
}

func setSessionCookie(w http.ResponseWriter, name, value string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     cookiePath,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearSessionCookies(w http.ResponseWriter) {
	setSessionCookie(w, sessionCookie, "", -time.Second)
	setSessionCookie(w, refreshCookie, "", -time.Second)
}

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
	// the deploy wrote them (the scope is the deploy's to grant, never this plugin's).
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

// signOut revokes the refresh token, best effort, clears the session and answers the hosted
// sign-out URL, which the web app navigates to so Cognito's own cookie is cleared too. Without
// that, the next sign-in would skip the password.
func (a *hostedAuth) signOut(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookie); err == nil && c.Value != "" {
		_ = a.revoke(r.Context(), c.Value)
	}
	clearSessionCookies(w)
	apikit.WriteJSON(w, http.StatusOK, map[string]string{"logout_url": a.logoutURL})
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

type hostedTokens struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// tokenError is a failed call. kind is the OAuth error code (invalid_grant), or empty when the
// call never got a usable answer; status is the HTTP status when there was one.
type tokenError struct {
	kind   string
	status int
	err    error
}

func (e *tokenError) Error() string {
	switch {
	case e.err != nil:
		return fmt.Sprintf("token endpoint unreachable: %v", e.err)
	case e.kind != "":
		return fmt.Sprintf("token endpoint answered %d %s", e.status, e.kind)
	default:
		return fmt.Sprintf("token endpoint answered %d", e.status)
	}
}

func (e *tokenError) Unwrap() error { return e.err }

// unreachable is a failure that is the platform's, not the person's: no answer, or a 5xx. It does
// not end a session.
func (e *tokenError) unreachable() bool { return e.err != nil || e.status >= 500 }

// tokenRequest posts a form to the hosted token endpoint. The client is public: there is no secret
// and no Authorization header.
func (a *hostedAuth) tokenRequest(ctx context.Context, form url.Values) (hostedTokens, *tokenError) {
	ctx, cancel := context.WithTimeout(ctx, hostedTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return hostedTokens{}, &tokenError{err: err}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return hostedTokens{}, &tokenError{err: err}
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTokenBytes))
	if err != nil {
		return hostedTokens{}, &tokenError{err: err}
	}
	var out struct {
		hostedTokens
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &out)
	if resp.StatusCode != http.StatusOK {
		return hostedTokens{}, &tokenError{kind: out.Error, status: resp.StatusCode}
	}
	if out.IDToken == "" {
		return hostedTokens{}, &tokenError{kind: "no_id_token", status: resp.StatusCode, err: errors.New("the answer holds no id_token")}
	}
	return out.hostedTokens, nil
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
