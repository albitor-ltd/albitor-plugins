package shell

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"webapp/internal/apikit"
	"webapp/internal/identity"
	"webapp/internal/localauth"
)

// The Cognito half of the identity seam: where the pool comes from, the calls the API makes to
// Cognito, and the session cookies.
//
// The API makes those calls, not the browser. The page's Content-Security-Policy allows
// connections to its own origin only (connect-src 'self'), and the web app is a static build that
// carries no pool ids. Tokens travel as HttpOnly cookies the page's JavaScript cannot read, which
// is the hardened shape anyway: no token in localStorage for an injected script to steal.

// The environment the deploy target hands the function (services.auth in
// albitor.yaml): the pool, its public client, and the region.
const (
	envUserPoolID = "AUTH_USER_POOL_ID"
	envClientID   = "AUTH_CLIENT_ID"
	envRegion     = "AWS_REGION"
	// envSignUp is services.auth.sign_up: "invite_only" or "open". Unset means "open", so an app
	// deployed before the variable existed behaves as it always did.
	envSignUp = "AUTH_SIGN_UP"
	// envAdminGroup names the group whose members may manage people (invite_only only).
	envAdminGroup = "AUTH_ADMIN_GROUP"
	// envUserAdminGroup names the dedicated group that may manage people (invite_only only). It is
	// not AUTH_ADMIN_GROUP, which the platform's demo and verifier accounts also belong to. Empty
	// means nobody may: the People endpoints fail closed.
	envUserAdminGroup = "AUTH_USER_ADMIN_GROUP"
	envLambdaName     = "AWS_LAMBDA_FUNCTION_NAME"
	envLocalRun       = "ALBITOR_LOCAL"
)

// The session cookies. Both HttpOnly, Secure and SameSite=Strict: no script
// reads them and no other site's form sends them.
const (
	sessionCookie = "session"
	refreshCookie = "session_refresh"
	// refreshCookiePath keeps the refresh token off every request but the
	// API's: the gate refreshes an expired session on any /api/ call.
	refreshCookiePath = "/api"
	// sessionMaxAge matches the public client's refresh-token validity.
	sessionMaxAge = 30 * 24 * time.Hour
)

const cognitoTimeout = 10 * time.Second

// signIn is how this process answers who a request is, decided once from the
// environment: a pool, the local issuer, nobody (a local run with no sign-in),
// or a failure, which closes every route.
type signIn struct {
	auth      *cognitoAuth
	anonymous bool
	err       error
}

// signInFromEnv decides where identity comes from:
//
//   - AUTH_USER_POOL_ID, AUTH_CLIENT_ID and AWS_REGION: that pool. Any of the
//     first two set without the full, well-formed trio is a failure, never a
//     fallback.
//   - a deployed function (AWS_LAMBDA_FUNCTION_NAME) with no pool: a failure.
//   - a local run (ALBITOR_LOCAL=1) with ALBITOR_LOCAL_AUTH=1: the local issuer
//     api-go starts, alone. The sign-in calls say they are not available.
//   - a local run with neither: nobody, and every request acts as the
//     anonymous owner.
//   - anything else: a failure. An API that is neither local nor configured
//     refuses everything rather than guess.
//
// The pool and the deployed marker are decided before the local issuer, so a
// named pool or a deployed function never trusts it, whatever else is set.
func signInFromEnv(getenv func(string) string) signIn {
	pool := strings.TrimSpace(getenv(envUserPoolID))
	client := strings.TrimSpace(getenv(envClientID))
	if pool != "" || client != "" {
		cfg := cognitoConfig{
			Region: strings.TrimSpace(getenv(envRegion)), UserPoolID: pool, ClientID: client,
			SignUp: strings.TrimSpace(getenv(envSignUp)), AdminGroup: strings.TrimSpace(getenv(envAdminGroup)),
			UserAdminGroup: strings.TrimSpace(getenv(envUserAdminGroup)),
		}
		if !cfg.valid() {
			return signIn{err: fmt.Errorf("%s, %s and %s must all be set, and well formed", envUserPoolID, envClientID, envRegion)}
		}
		if cfg.SignUp != "" && cfg.SignUp != signUpOpen && cfg.SignUp != signUpInviteOnly {
			return signIn{err: fmt.Errorf("%s must be %q or %q", envSignUp, signUpInviteOnly, signUpOpen)}
		}
		return signIn{auth: newCognitoAuth(cfg)}
	}
	if strings.TrimSpace(getenv(envLambdaName)) != "" {
		return signIn{err: fmt.Errorf("a deployed function has no user pool: %s and %s are not set", envUserPoolID, envClientID)}
	}
	if strings.TrimSpace(getenv(envLocalRun)) != "1" {
		return signIn{err: errors.New("no user pool is configured and this is not a local run (ALBITOR_LOCAL=1)")}
	}
	if strings.TrimSpace(getenv(localauth.EnabledEnv)) == "1" {
		return signIn{auth: newLocalIssuerAuth(localauth.IssuerURL(getenv("PORT")))}
	}
	return signIn{anonymous: true}
}

// cognitoConfig names the pool and client the API trusts.
type cognitoConfig struct {
	Region     string
	UserPoolID string
	ClientID   string
	// SignUp is "invite_only", "open" or empty (open).
	SignUp string
	// AdminGroup is the app's admin group (AUTH_ADMIN_GROUP).
	AdminGroup string
	// UserAdminGroup is the group that manages people in an invite_only app.
	UserAdminGroup string
}

// The two sign-up modes (services.auth.sign_up).
const (
	signUpOpen       = "open"
	signUpInviteOnly = "invite_only"
)

func (c cognitoConfig) valid() bool {
	return apikit.IsSafeToken(c.Region, 32) && apikit.IsSafeToken(c.UserPoolID, 128) && apikit.IsSafeToken(c.ClientID, 128) &&
		(c.AdminGroup == "" || apikit.IsSafeToken(c.AdminGroup, 128)) &&
		(c.UserAdminGroup == "" || apikit.IsSafeToken(c.UserAdminGroup, 128))
}

func (c cognitoConfig) issuer() string {
	return "https://cognito-idp." + c.Region + ".amazonaws.com/" + c.UserPoolID
}

func (c cognitoConfig) endpoint() string {
	return "https://cognito-idp." + c.Region + ".amazonaws.com/"
}

// cognitoAuth is one pool and client: the token check and the public calls.
// With local set it is api-go's local issuer instead: tokens are checked the
// same way, and there is no pool to make the sign-in calls to.
type cognitoAuth struct {
	clientID string
	endpoint string
	client   *http.Client
	verifier *cognitoVerifier
	local    bool

	// signUp is the pool's sign-up mode; the zero value is open.
	signUp         string
	adminGroup     string
	userAdminGroup string
	poolID         string
	region         string
}

// inviteOnly reports whether the pool is invite-only: self sign-up is off, accounts are created
// by an administrator, and the People endpoints are live.
func (a *cognitoAuth) inviteOnly() bool { return !a.local && a.signUp == signUpInviteOnly }

// signUpMode is what GET /api/auth/config reports.
func (a *cognitoAuth) signUpMode() string {
	if a.inviteOnly() {
		return signUpInviteOnly
	}
	return signUpOpen
}

// newLocalIssuerAuth trusts the local issuer at issuer, and nothing else.
func newLocalIssuerAuth(issuer string) *cognitoAuth {
	client := &http.Client{Timeout: cognitoTimeout}
	return &cognitoAuth{
		clientID: localauth.ClientID,
		client:   client,
		verifier: newCognitoVerifier(issuer, localauth.ClientID, issuer+"/.well-known/jwks.json", client),
		local:    true,
	}
}

func newCognitoAuth(cfg cognitoConfig) *cognitoAuth {
	client := &http.Client{Timeout: cognitoTimeout}
	return &cognitoAuth{
		clientID: cfg.ClientID,
		endpoint: cfg.endpoint(),
		client:   client,
		verifier: newCognitoVerifier(cfg.issuer(), cfg.ClientID, cfg.issuer()+"/.well-known/jwks.json", client),

		signUp:         cfg.SignUp,
		adminGroup:     cfg.AdminGroup,
		userAdminGroup: cfg.UserAdminGroup,
		poolID:         cfg.UserPoolID,
		region:         cfg.Region,
	}
}

// authenticate returns the verified identity a request carries: from its
// bearer token, or from the session cookie, refreshing an expired session from
// the refresh cookie. On a refusal it has already written the response.
func (a *cognitoAuth) authenticate(w http.ResponseWriter, r *http.Request, logger *slog.Logger) (identity.Identity, bool) {
	raw, fromCookie := requestToken(r)
	if raw == "" {
		writeUnauthenticated(w)
		return identity.Identity{}, false
	}
	id, err := a.verifier.verify(r.Context(), raw)
	if errors.Is(err, errTokenExpired) && fromCookie {
		id, err = a.refreshSession(w, r)
	}
	switch {
	case err == nil:
		return id, true
	case errors.Is(err, errVerifierUnavailable):
		logger.ErrorContext(r.Context(), "could not check a sign-in token",
			slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())),
			slog.String("error", err.Error()))
		apikit.WriteError(w, http.StatusServiceUnavailable, "sign-in is unavailable")
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

func writeUnauthenticated(w http.ResponseWriter) {
	apikit.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "sign-in required", "code": "unauthenticated"})
}

// requestToken returns the request's token and whether it came from the
// session cookie. A bearer header wins.
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

// refreshSession swaps the refresh cookie for a new session.
func (a *cognitoAuth) refreshSession(w http.ResponseWriter, r *http.Request) (identity.Identity, error) {
	c, err := r.Cookie(refreshCookie)
	if a.local || err != nil || c.Value == "" {
		return identity.Identity{}, errTokenInvalid
	}
	tokens, cerr := a.initiateAuth(r.Context(), "REFRESH_TOKEN_AUTH", map[string]string{"REFRESH_TOKEN": c.Value})
	if cerr != nil {
		if cerr.kind == "" {
			return identity.Identity{}, fmt.Errorf("%w: %w", errVerifierUnavailable, cerr)
		}
		return identity.Identity{}, errTokenInvalid
	}
	id, err := a.verifier.verify(r.Context(), tokens.IDToken)
	if err != nil {
		return identity.Identity{}, err
	}
	setSessionCookie(w, sessionCookie, tokens.IDToken, "/", sessionMaxAge)
	return id, nil
}

func setSessionCookie(w http.ResponseWriter, name, value, path string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearSessionCookies(w http.ResponseWriter) {
	setSessionCookie(w, sessionCookie, "", "/", -time.Second)
	setSessionCookie(w, refreshCookie, "", refreshCookiePath, -time.Second)
}

type cognitoTokens struct {
	IDToken      string `json:"IdToken"`
	AccessToken  string `json:"AccessToken"`
	RefreshToken string `json:"RefreshToken"`
}

func (a *cognitoAuth) initiateAuth(ctx context.Context, flow string, params map[string]string) (cognitoTokens, *cognitoError) {
	var out struct {
		ChallengeName        string        `json:"ChallengeName"`
		Session              string        `json:"Session"`
		AuthenticationResult cognitoTokens `json:"AuthenticationResult"`
	}
	if cerr := a.call(ctx, "InitiateAuth", map[string]any{
		"ClientId": a.clientID, "AuthFlow": flow, "AuthParameters": params,
	}, &out); cerr != nil {
		return cognitoTokens{}, cerr
	}
	return challengeOrTokens(out.ChallengeName, out.Session, out.AuthenticationResult)
}

// The challenge a person invited by the app's owner meets on first sign-in,
// and the error kinds that stand for it and for every other challenge.
const (
	challengeNewPassword     = "NEW_PASSWORD_REQUIRED"
	kindNewPasswordRequired  = "NewPasswordRequired"
	kindChallengeRequired    = "ChallengeRequired"
	maxChallengeSessionBytes = 8 << 10
)

// challengeOrTokens turns a sign-in answer into tokens, or into the error that
// says what the pool wants instead. A temporary password to replace
// (NEW_PASSWORD_REQUIRED, an invited person's first sign-in) is a step the
// sign-in screens handle; any other challenge (MFA) needs a screen of its own,
// so the app adds it when it turns one on.
func challengeOrTokens(name, session string, tokens cognitoTokens) (cognitoTokens, *cognitoError) {
	switch {
	case name == challengeNewPassword && session != "":
		return cognitoTokens{}, &cognitoError{kind: kindNewPasswordRequired, message: name, session: session}
	case name != "" || tokens.IDToken == "":
		return cognitoTokens{}, &cognitoError{kind: kindChallengeRequired, message: name}
	}
	return tokens, nil
}

// cognitoError is a failed call. kind is Cognito's exception name, or empty
// when the call never got an answer from Cognito.
type cognitoError struct {
	kind    string
	message string
	// session is the challenge session of a kindNewPasswordRequired answer.
	session string
	err     error
}

func (e *cognitoError) Error() string {
	if e.kind == "" {
		return fmt.Sprintf("cognito unreachable: %v", e.err)
	}
	return e.kind + ": " + e.message
}

func (e *cognitoError) Unwrap() error { return e.err }

// call makes one of Cognito's public user-pool operations. They need no AWS
// credentials: the app client id is the whole of the authority.
func (a *cognitoAuth) call(ctx context.Context, op string, in any, out any) *cognitoError {
	return a.send(ctx, op, in, out, nil)
}

// send posts one operation. sign, when set, signs the request before it is sent; the public
// operations pass nil.
func (a *cognitoAuth) send(ctx context.Context, op string, in any, out any, sign func(req *http.Request, payload []byte) error) *cognitoError {
	payload, err := json.Marshal(in)
	if err != nil {
		return &cognitoError{err: err}
	}
	ctx, cancel := context.WithTimeout(ctx, cognitoTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(payload))
	if err != nil {
		return &cognitoError{err: err}
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService."+op)
	if sign != nil {
		if err := sign(req, payload); err != nil {
			return &cognitoError{err: err}
		}
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return &cognitoError{err: err}
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return &cognitoError{err: err}
	}
	if resp.StatusCode != http.StatusOK {
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
			return &cognitoError{err: fmt.Errorf("%s answered %d", op, resp.StatusCode)}
		}
		return &cognitoError{kind: kind, message: e.Message}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return &cognitoError{err: fmt.Errorf("decode %s: %w", op, err)}
	}
	return nil
}
