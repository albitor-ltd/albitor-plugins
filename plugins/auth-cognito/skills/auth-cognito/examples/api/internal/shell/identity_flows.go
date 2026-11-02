package shell

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"webapp/internal/apikit"
)

// The sign-in calls the web screens make: POST /api/auth/sign-in, sign-up, confirm, resend,
// forgot, reset, new-password and sign-out. They carry no session (they are how a person gets
// one), so IdentityVerify lets everything under /api/auth/ through. The signed-in calls
// (identity_people.go) sit under the same prefix.
//
// registerAuthRoutes is called from registerIdentityRoutes in identity_verify.go.

const maxAuthBodyBytes = 16 << 10

// registerAuthRoutes answers /api/auth/*: the sign-in calls, and in a local run, which has no
// pool, a pointer to the local issuer.
func registerAuthRoutes(mux *http.ServeMux, setup signIn, logger *slog.Logger) {
	mux.HandleFunc(authPathRoot, func(w http.ResponseWriter, r *http.Request) {
		if setup.auth == nil {
			if r.URL.Path == authPathRoot+"config" && r.Method == http.MethodGet {
				// No pool, so no invitations: the screens offer what a local run always has.
				apikit.WriteJSON(w, http.StatusOK, map[string]string{"sign_up": signUpOpen})
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

// serveAuth answers POST /api/auth/<action>: the calls the sign-in, sign-up,
// confirm-code and reset screens make. Each takes a small JSON body.
func (a *cognitoAuth) serveAuth(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	// The routes that are not one of the public sign-in calls: the mode, a signed-in person's own
	// password, and the People admin (identity_people.go).
	if a.serveSignedInRoute(w, r, logger) {
		return
	}
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
	// In an invite-only app nobody registers or recovers an account for themselves: an administrator
	// adds people and resets passwords, so these calls do not exist.
	if a.inviteOnly() && selfServiceActions[strings.TrimPrefix(r.URL.Path, authPathRoot)] {
		writeAuthError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	// JSON only. Another site's HTML form cannot send this content type
	// without the browser asking first, so it cannot sign a visitor in to an
	// account of its choosing.
	if !isJSONRequest(r) {
		writeAuthError(w, http.StatusUnsupportedMediaType, "invalid_input", "send application/json")
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Code     string `json:"code"`
		Session  string `json:"session"`
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
	case "new-password":
		// The second half of a first sign-in: an invited person signed in with
		// the temporary password the pool emailed and was asked for their own.
		var tokens cognitoTokens
		tokens, cerr = a.respondNewPassword(r.Context(), email, body.Password, body.Session)
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

// selfServiceActions are the calls only a pool that lets people register themselves has.
var selfServiceActions = map[string]bool{"sign-up": true, "confirm": true, "resend": true, "forgot": true, "reset": true}

func isJSONRequest(r *http.Request) bool {
	mt, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
	return strings.EqualFold(strings.TrimSpace(mt), "application/json")
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

// respondNewPassword answers a NEW_PASSWORD_REQUIRED challenge with the
// person's own password. The session is the opaque token the pool handed back
// with the challenge; it is bound to this user and lasts a few minutes, so
// holding it in the page between the two calls gives nobody anything the
// temporary password did not.
func (a *cognitoAuth) respondNewPassword(ctx context.Context, email, password, session string) (cognitoTokens, *cognitoError) {
	if password == "" || session == "" || len(session) > maxChallengeSessionBytes {
		return cognitoTokens{}, &cognitoError{kind: "InvalidParameterException", message: "a new password and the sign-in session are required"}
	}
	var out struct {
		ChallengeName        string        `json:"ChallengeName"`
		Session              string        `json:"Session"`
		AuthenticationResult cognitoTokens `json:"AuthenticationResult"`
	}
	if cerr := a.call(ctx, "RespondToAuthChallenge", map[string]any{
		"ClientId":      a.clientID,
		"ChallengeName": challengeNewPassword,
		"Session":       session,
		"ChallengeResponses": map[string]string{
			"USERNAME":     email,
			"NEW_PASSWORD": password,
		},
	}, &out); cerr != nil {
		return cognitoTokens{}, cerr
	}
	// Asked for the same step again would loop the screen: only tokens succeed.
	if out.ChallengeName == challengeNewPassword {
		return cognitoTokens{}, &cognitoError{kind: kindChallengeRequired, message: out.ChallengeName}
	}
	return challengeOrTokens(out.ChallengeName, out.Session, out.AuthenticationResult)
}

// writeCognitoError turns Cognito's answer into one the screens can act on.
// The code is stable; the message is safe to show.
func (a *cognitoAuth) writeCognitoError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, action string, e *cognitoError) {
	switch e.kind {
	case "NotAuthorizedException", "UserNotFoundException":
		if action == "change-password" {
			writeAuthError(w, http.StatusUnauthorized, "invalid_credentials", "your current password is not right")
			return
		}
		if action == "new-password" {
			// The challenge session lapsed (a few minutes) or was never this person's.
			writeAuthError(w, http.StatusUnauthorized, "session_expired", "that sign-in has expired: sign in again")
			return
		}
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
	case kindNewPasswordRequired:
		apikit.WriteJSON(w, http.StatusConflict, map[string]string{
			"error": "choose a new password to finish signing in", "code": "new_password_required", "session": e.session,
		})
	case kindChallengeRequired:
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
