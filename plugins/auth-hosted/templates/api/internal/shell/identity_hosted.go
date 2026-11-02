package shell

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"webapp/internal/apikit"
	"webapp/internal/identity"
	"webapp/internal/localauth"
)

// The hosted half of the identity seam: where the user pool comes from, the token endpoint, and
// the session cookies.
//
// The API runs the sign-in flow, not the browser. The page is a static build that carries no pool
// ids, and its Content-Security-Policy allows connections to its own origin only. The tokens
// become HttpOnly cookies the page's JavaScript cannot read, so there is no token in localStorage
// for an injected script to steal, and the app never sees a password.

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

const (
	maxTokenBytes = 1 << 20
	hostedTimeout = 10 * time.Second
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
//   - a local run (ALBITOR_LOCAL=1) with ALBITOR_LOCAL_AUTH=1: the local issuer the API starts,
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

// hostedAuth is one pool and client: the token check and the hosted-sign-in flow. With local set
// it is the local issuer instead: tokens are checked the same way, and there is no pool to
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

// platformSeededDomains are the domains of the accounts the platform seeds into every pool (the
// verifier and the demo account). They are .invalid (RFC 2606), so no real person can hold one, and
// SIGNUP_DOMAINS never locks them out: journeys and the published service's demo sign-in need them.
var platformSeededDomains = []string{"albitor-verify.invalid", "albitor-demo.invalid"}

// platformSeededEmail reports whether an already lower-cased address is on a platform-seeded
// domain. Only the part after the last @ counts, so lookalikes and extra @ parts do not match.
func platformSeededEmail(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 1 {
		return false
	}
	return slices.Contains(platformSeededDomains, email[at+1:])
}

// acceptIDToken is the one rule for who may hold a session, applied to every ID token the API
// takes: at the callback, on a refresh, and on each request. It must be an ID token, for an address
// that Cognito has verified, which SIGNUP_DOMAINS allows (the platform-seeded domains always pass
// it). A person can change their own address to another verified one, so it is the token's address,
// every time (a string comparison, no call).
func (a *hostedAuth) acceptIDToken(c jwtClaims) error {
	if c.TokenUse != "id" {
		return errTokenInvalid
	}
	email := strings.ToLower(strings.TrimSpace(c.Email))
	if email == "" || !bool(c.EmailVerified) {
		return errEmailUnverified
	}
	if !platformSeededEmail(email) && !signupAllowed(a.domains, email) {
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
