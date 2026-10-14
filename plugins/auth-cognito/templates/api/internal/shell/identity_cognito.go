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

// The Cognito half of sign-in: where the pool comes from, the sign-in calls the
// web app makes (POST /api/auth/*), and the session cookies.
//
// WHY THE API MAKES THE SIGN-IN CALLS, NOT THE BROWSER. The page's
// Content-Security-Policy allows connections to its own origin only
// (connect-src 'self'), and the web app is a static build that carries no pool
// ids. So the browser posts to /api/auth/*, and the API calls Cognito's public,
// unauthenticated operations on its behalf. The tokens come back as HttpOnly
// cookies the page's JavaScript cannot read, which is the hardened shape
// anyway: no token in localStorage for an injected script to steal.

// The environment the deploy target hands the function (services.auth in
// albitor.yaml): the pool, its public client, and the region.
const (
	envUserPoolID = "AUTH_USER_POOL_ID"
	envClientID   = "AUTH_CLIENT_ID"
	envRegion     = "AWS_REGION"
	envLambdaName = "AWS_LAMBDA_FUNCTION_NAME"
	envLocalRun   = "ALBITOR_LOCAL"
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

const (
	maxAuthBodyBytes = 16 << 10
	cognitoTimeout   = 10 * time.Second
)

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
		cfg := cognitoConfig{Region: strings.TrimSpace(getenv(envRegion)), UserPoolID: pool, ClientID: client}
		if !cfg.valid() {
			return signIn{err: fmt.Errorf("%s, %s and %s must all be set, and well formed", envUserPoolID, envClientID, envRegion)}
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
}

func (c cognitoConfig) valid() bool {
	return apikit.IsSafeToken(c.Region, 32) && apikit.IsSafeToken(c.UserPoolID, 128) && apikit.IsSafeToken(c.ClientID, 128)
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
		if fromCookie {
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

// serveAuth answers POST /api/auth/<action>: the calls the sign-in, sign-up,
// confirm-code and reset screens make. Each takes a small JSON body.
func (a *cognitoAuth) serveAuth(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	if r.Method != http.MethodPost {
		apikit.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	if a.local {
		// A local run has no pool to sign in to. A spec signs in through the
		// local issuer instead (web/tests/helpers/local-auth.ts).
		apikit.WriteJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "sign-in is not available in a local run: use the local issuer", "code": "local",
		})
		return
	}
	// JSON only. Another site's HTML form cannot send this content type
	// without the browser asking first, so it cannot sign a visitor in to an
	// account of its choosing.
	if mt, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";"); !strings.EqualFold(strings.TrimSpace(mt), "application/json") {
		writeAuthError(w, http.StatusUnsupportedMediaType, "invalid_input", "send application/json")
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodyBytes)).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeAuthError(w, http.StatusBadRequest, "invalid_input", "invalid json")
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	code := strings.TrimSpace(body.Code)

	action := strings.TrimPrefix(r.URL.Path, authPathRoot)
	if action != "sign-out" && email == "" {
		writeAuthError(w, http.StatusBadRequest, "invalid_input", "email is required")
		return
	}

	var cerr *cognitoError
	switch action {
	case "sign-in":
		var tokens cognitoTokens
		tokens, cerr = a.initiateAuth(r.Context(), "USER_PASSWORD_AUTH", map[string]string{"USERNAME": email, "PASSWORD": body.Password})
		if cerr == nil {
			a.finishSignIn(w, r, tokens, logger)
			return
		}
	case "sign-up":
		var out struct {
			UserConfirmed bool `json:"UserConfirmed"`
		}
		cerr = a.call(r.Context(), "SignUp", map[string]any{
			"ClientId": a.clientID, "Username": email, "Password": body.Password,
			"UserAttributes": []map[string]string{{"Name": "email", "Value": email}},
		}, &out)
		if cerr == nil {
			apikit.WriteJSON(w, http.StatusCreated, map[string]bool{"confirmation_required": !out.UserConfirmed})
			return
		}
	case "confirm":
		cerr = a.call(r.Context(), "ConfirmSignUp", map[string]any{
			"ClientId": a.clientID, "Username": email, "ConfirmationCode": code,
		}, nil)
	case "resend":
		cerr = a.call(r.Context(), "ResendConfirmationCode", map[string]any{"ClientId": a.clientID, "Username": email}, nil)
	case "forgot":
		cerr = a.call(r.Context(), "ForgotPassword", map[string]any{"ClientId": a.clientID, "Username": email}, nil)
	case "reset":
		cerr = a.call(r.Context(), "ConfirmForgotPassword", map[string]any{
			"ClientId": a.clientID, "Username": email, "ConfirmationCode": code, "Password": body.Password,
		}, nil)
	case "sign-out":
		a.signOut(w, r)
		return
	default:
		apikit.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	if cerr != nil {
		a.writeCognitoError(w, r, logger, action, cerr)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// finishSignIn checks the new ID token like any other and sets the session.
func (a *cognitoAuth) finishSignIn(w http.ResponseWriter, r *http.Request, tokens cognitoTokens, logger *slog.Logger) {
	id, err := a.verifier.verify(r.Context(), tokens.IDToken)
	if err != nil {
		logger.ErrorContext(r.Context(), "the pool issued a token this API does not accept",
			slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())),
			slog.String("error", err.Error()))
		writeAuthError(w, http.StatusBadGateway, "unavailable", "sign-in is unavailable")
		return
	}
	setSessionCookie(w, sessionCookie, tokens.IDToken, "/", sessionMaxAge)
	if tokens.RefreshToken != "" {
		setSessionCookie(w, refreshCookie, tokens.RefreshToken, refreshCookiePath, sessionMaxAge)
	}
	apikit.WriteJSON(w, http.StatusOK, id)
}

// signOut revokes the refresh token, best effort, and clears the session.
func (a *cognitoAuth) signOut(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookie); err == nil && c.Value != "" {
		_ = a.call(r.Context(), "RevokeToken", map[string]any{"ClientId": a.clientID, "Token": c.Value}, nil)
	}
	clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

type cognitoTokens struct {
	IDToken      string `json:"IdToken"`
	AccessToken  string `json:"AccessToken"`
	RefreshToken string `json:"RefreshToken"`
}

func (a *cognitoAuth) initiateAuth(ctx context.Context, flow string, params map[string]string) (cognitoTokens, *cognitoError) {
	var out struct {
		ChallengeName        string        `json:"ChallengeName"`
		AuthenticationResult cognitoTokens `json:"AuthenticationResult"`
	}
	if cerr := a.call(ctx, "InitiateAuth", map[string]any{
		"ClientId": a.clientID, "AuthFlow": flow, "AuthParameters": params,
	}, &out); cerr != nil {
		return cognitoTokens{}, cerr
	}
	if out.ChallengeName != "" || out.AuthenticationResult.IDToken == "" {
		// A challenge (a temporary password, MFA) needs a screen this plugin
		// does not ship. Add it when the app turns either on.
		return cognitoTokens{}, &cognitoError{kind: "ChallengeRequired", message: out.ChallengeName}
	}
	return out.AuthenticationResult, nil
}

// cognitoError is a failed call. kind is Cognito's exception name, or empty
// when the call never got an answer from Cognito.
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

// call makes one of Cognito's public user-pool operations. They need no AWS
// credentials: the app client id is the whole of the authority.
func (a *cognitoAuth) call(ctx context.Context, op string, in any, out any) *cognitoError {
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

// writeCognitoError turns Cognito's answer into one the screens can act on.
// The code is stable; the message is safe to show.
func (a *cognitoAuth) writeCognitoError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, action string, e *cognitoError) {
	switch e.kind {
	case "NotAuthorizedException", "UserNotFoundException":
		writeAuthError(w, http.StatusUnauthorized, "invalid_credentials", "incorrect email or password")
	case "UserNotConfirmedException":
		writeAuthError(w, http.StatusForbidden, "not_confirmed", "confirm your email address first")
	case "UsernameExistsException", "AliasExistsException":
		writeAuthError(w, http.StatusConflict, "exists", "an account with that email already exists")
	case "InvalidPasswordException":
		writeAuthError(w, http.StatusBadRequest, "invalid_password", passwordMessage(e.message))
	case "CodeMismatchException":
		writeAuthError(w, http.StatusBadRequest, "invalid_code", "that code is not right")
	case "ExpiredCodeException":
		writeAuthError(w, http.StatusBadRequest, "expired_code", "that code has expired: ask for a new one")
	case "InvalidParameterException":
		writeAuthError(w, http.StatusBadRequest, "invalid_input", "check the details and try again")
	case "LimitExceededException", "TooManyRequestsException", "TooManyFailedAttemptsException", "CodeDeliveryFailureException":
		writeAuthError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts: wait a while and try again")
	case "ChallengeRequired":
		writeAuthError(w, http.StatusConflict, "challenge_unsupported", "this account needs a sign-in step the app does not have yet")
	default:
		logger.ErrorContext(r.Context(), "sign-in call failed",
			slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())),
			slog.String("action", action),
			slog.String("error", e.Error()))
		writeAuthError(w, http.StatusBadGateway, "unavailable", "sign-in is unavailable: try again shortly")
	}
}

// passwordMessage keeps Cognito's description of the policy a password missed,
// which names the rule and nothing about the account.
func passwordMessage(msg string) string {
	if _, rule, ok := strings.Cut(msg, ": "); ok && rule != "" {
		return "password does not meet the policy: " + rule
	}
	return "password does not meet the policy"
}

func writeAuthError(w http.ResponseWriter, status int, code, msg string) {
	apikit.WriteJSON(w, status, map[string]string{"error": msg, "code": code})
}
