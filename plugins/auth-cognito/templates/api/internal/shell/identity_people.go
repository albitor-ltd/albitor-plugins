package shell

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"sync"
	"time"

	"webapp/internal/apikit"
	"webapp/internal/identity"
)

// THE SIGNED-IN HALF OF /api/auth/*. The sign-in calls in identity_cognito.go carry no session
// (they are how a person gets one). These carry one:
//
//   - GET  /api/auth/config           public: the sign-up mode, so the screens know what to offer.
//   - POST /api/auth/change-password  any signed-in person, in either mode.
//   - the People admin (GET/POST /api/auth/users, POST /api/auth/users/{disable,enable,
//     reset-password}): invite_only only, and only for the admin group. In an open app they answer
//     404, exactly as an unknown route does.
//
// The People calls are the pool's administrator operations. They are signed with the function's
// execution-role credentials, which the deploy target grants on this one pool and only when the
// app is invite-only. A temporary password is generated here, returned once in the response and
// never logged or stored.

const (
	maxPeople        = 60
	groupLookupLimit = 8
)

// serveSignedInRoute answers the routes above and reports whether r was one of them.
func (a *cognitoAuth) serveSignedInRoute(w http.ResponseWriter, r *http.Request, logger *slog.Logger) bool {
	switch strings.TrimPrefix(r.URL.Path, authPathRoot) {
	case "config":
		if r.Method != http.MethodGet {
			writeAuthError(w, http.StatusNotFound, "not_found", "not found")
			return true
		}
		w.Header().Set("Cache-Control", "no-store")
		apikit.WriteJSON(w, http.StatusOK, map[string]string{"sign_up": a.signUpMode()})
	case "change-password":
		a.serveChangePassword(w, r, logger)
	case "users":
		switch r.Method {
		case http.MethodGet:
			a.servePeopleList(w, r, logger)
		case http.MethodPost:
			a.servePeopleCreate(w, r, logger)
		default:
			writeAuthError(w, http.StatusNotFound, "not_found", "not found")
		}
	case "users/disable":
		a.servePeopleToggle(w, r, logger, "AdminDisableUser", "disable")
	case "users/enable":
		a.servePeopleToggle(w, r, logger, "AdminEnableUser", "enable")
	case "users/reset-password":
		a.servePeopleResetPassword(w, r, logger)
	default:
		return false
	}
	return true
}

// ---- change password ----

// serveChangePassword is POST /api/auth/change-password {current_password, new_password}. The
// person is the session's; the current password is checked by signing in with it, which also
// yields the access token Cognito's ChangePassword needs (the session cookie holds only the ID
// token).
func (a *cognitoAuth) serveChangePassword(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	if r.Method != http.MethodPost {
		writeAuthError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if a.local {
		apikit.WriteJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "sign-in is not available in a local run: use the local issuer", "code": "local",
		})
		return
	}
	if !isJSONRequest(r) {
		writeAuthError(w, http.StatusUnsupportedMediaType, "invalid_input", "send application/json")
		return
	}
	id, ok := a.authenticate(w, r, logger)
	if !ok {
		return
	}
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if id.Email == "" || body.CurrentPassword == "" || body.NewPassword == "" {
		writeAuthError(w, http.StatusBadRequest, "invalid_input", "your current and new passwords are required")
		return
	}
	const action = "change-password"
	tokens, cerr := a.initiateAuth(r.Context(), "USER_PASSWORD_AUTH", map[string]string{"USERNAME": id.Email, "PASSWORD": body.CurrentPassword})
	if cerr != nil {
		a.writeCognitoError(w, r, logger, action, cerr)
		return
	}
	if cerr := a.call(r.Context(), "ChangePassword", map[string]any{
		"AccessToken": tokens.AccessToken, "PreviousPassword": body.CurrentPassword, "ProposedPassword": body.NewPassword,
	}, nil); cerr != nil {
		a.writeCognitoError(w, r, logger, action, cerr)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- People ----

// requireAdmin lets an invite_only app's admin through. On a refusal it has answered: 404 when
// the app is not invite-only (the endpoints do not exist), 401 with no session, 403 for anyone
// outside the admin group.
func (a *cognitoAuth) requireAdmin(w http.ResponseWriter, r *http.Request, logger *slog.Logger) (identity.Identity, bool) {
	if !a.inviteOnly() {
		writeAuthError(w, http.StatusNotFound, "not_found", "not found")
		return identity.Identity{}, false
	}
	id, ok := a.authenticate(w, r, logger)
	if !ok {
		return identity.Identity{}, false
	}
	if a.adminGroup == "" || !id.InGroup(a.adminGroup) {
		writeAuthError(w, http.StatusForbidden, "forbidden", "only an administrator can manage people")
		return identity.Identity{}, false
	}
	return id, true
}

// cognitoUser is the part of a ListUsers entry the list reads.
type cognitoUser struct {
	Username       string  `json:"Username"`
	Enabled        bool    `json:"Enabled"`
	UserStatus     string  `json:"UserStatus"`
	UserCreateDate float64 `json:"UserCreateDate"`
	Attributes     []struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	} `json:"Attributes"`
}

// person is one row of the People list.
type person struct {
	Email     string `json:"email"`
	Status    string `json:"status"`
	Enabled   bool   `json:"enabled"`
	Admin     bool   `json:"admin"`
	CreatedAt string `json:"created_at,omitempty"`
}

// servePeopleList is GET /api/auth/users[?next=<token>]: up to 60 people, with whether each is an
// administrator, and a token for the next page when there is one.
func (a *cognitoAuth) servePeopleList(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	if _, ok := a.requireAdmin(w, r, logger); !ok {
		return
	}
	in := map[string]any{"UserPoolId": a.poolID, "Limit": maxPeople}
	if next := r.URL.Query().Get("next"); next != "" {
		if len(next) > 4<<10 {
			writeAuthError(w, http.StatusBadRequest, "invalid_input", "check the details and try again")
			return
		}
		in["PaginationToken"] = next
	}
	var out struct {
		PaginationToken string        `json:"PaginationToken"`
		Users           []cognitoUser `json:"Users"`
	}
	if cerr := a.adminCall(r.Context(), "ListUsers", in, &out); cerr != nil {
		a.writePeopleError(w, r, logger, "list", cerr)
		return
	}
	people := make([]person, len(out.Users))
	for i, u := range out.Users {
		p := person{Email: u.Username, Status: u.UserStatus, Enabled: u.Enabled}
		for _, at := range u.Attributes {
			if at.Name == "email" && at.Value != "" {
				p.Email = at.Value
			}
		}
		if u.UserCreateDate > 0 {
			p.CreatedAt = time.Unix(int64(u.UserCreateDate), 0).UTC().Format(time.RFC3339)
		}
		people[i] = p
	}
	a.markAdmins(r, out.Users, people)
	w.Header().Set("Cache-Control", "no-store")
	apikit.WriteJSON(w, http.StatusOK, map[string]any{"people": people, "next": out.PaginationToken})
}

// markAdmins sets Admin on each person in the admin group, looking at a few people at a time. A
// lookup that fails leaves that person unmarked: the list is information, not a decision.
func (a *cognitoAuth) markAdmins(r *http.Request, users []cognitoUser, people []person) {
	if a.adminGroup == "" {
		return
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, groupLookupLimit)
	for i := range users {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			var out struct {
				Groups []struct {
					GroupName string `json:"GroupName"`
				} `json:"Groups"`
			}
			if cerr := a.adminCall(r.Context(), "AdminListGroupsForUser", map[string]any{
				"UserPoolId": a.poolID, "Username": users[i].Username,
			}, &out); cerr != nil {
				return
			}
			for _, g := range out.Groups {
				if g.GroupName == a.adminGroup {
					people[i].Admin = true
				}
			}
		}()
	}
	wg.Wait()
}

// servePeopleCreate is POST /api/auth/users {email, admin}: create the account with a generated
// temporary password and send nothing. The password is in the response, once; the administrator
// passes it on. The person chooses their own at first sign-in.
func (a *cognitoAuth) servePeopleCreate(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	if _, ok := a.requireAdmin(w, r, logger); !ok {
		return
	}
	var body struct {
		Email string `json:"email"`
		Admin bool   `json:"admin"`
	}
	if !a.peopleBody(w, r, &body) {
		return
	}
	email, ok := normaliseEmail(body.Email)
	if !ok {
		writeAuthError(w, http.StatusBadRequest, "invalid_input", "enter a valid email address")
		return
	}
	temp, err := generateTempPassword()
	if err != nil {
		logger.ErrorContext(r.Context(), "could not generate a temporary password",
			slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())))
		writeAuthError(w, http.StatusInternalServerError, "unavailable", "could not create the account: try again")
		return
	}
	if cerr := a.adminCall(r.Context(), "AdminCreateUser", map[string]any{
		"UserPoolId":        a.poolID,
		"Username":          email,
		"MessageAction":     "SUPPRESS",
		"TemporaryPassword": temp,
		"UserAttributes": []map[string]string{
			{"Name": "email", "Value": email},
			{"Name": "email_verified", "Value": "true"},
		},
	}, nil); cerr != nil {
		a.writePeopleError(w, r, logger, "create", cerr)
		return
	}
	admin := false
	if body.Admin && a.adminGroup != "" {
		if cerr := a.adminCall(r.Context(), "AdminAddUserToGroup", map[string]any{
			"UserPoolId": a.poolID, "Username": email, "GroupName": a.adminGroup,
		}, nil); cerr != nil {
			// The account exists and the password must still reach the administrator: report the
			// person as not (yet) an administrator rather than lose it.
			logger.ErrorContext(r.Context(), "could not add a new person to the admin group",
				slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())),
				slog.String("error", cerr.Error()))
		} else {
			admin = true
		}
	}
	writeTempPassword(w, http.StatusCreated, map[string]any{"email": email, "temporary_password": temp, "admin": admin})
}

// servePeopleToggle is POST /api/auth/users/{disable,enable} {email}.
func (a *cognitoAuth) servePeopleToggle(w http.ResponseWriter, r *http.Request, logger *slog.Logger, op, label string) {
	if r.Method != http.MethodPost {
		writeAuthError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	admin, ok := a.requireAdmin(w, r, logger)
	if !ok {
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if !a.peopleBody(w, r, &body) {
		return
	}
	email, ok := normaliseEmail(body.Email)
	if !ok {
		writeAuthError(w, http.StatusBadRequest, "invalid_input", "enter a valid email address")
		return
	}
	if op == "AdminDisableUser" && strings.EqualFold(email, admin.Email) {
		writeAuthError(w, http.StatusBadRequest, "invalid_input", "you cannot disable your own account")
		return
	}
	if cerr := a.adminCall(r.Context(), op, map[string]any{"UserPoolId": a.poolID, "Username": email}, nil); cerr != nil {
		a.writePeopleError(w, r, logger, label, cerr)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// servePeopleResetPassword is POST /api/auth/users/reset-password {email}: a new temporary
// password, returned once. The person must replace it at their next sign-in.
func (a *cognitoAuth) servePeopleResetPassword(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	if r.Method != http.MethodPost {
		writeAuthError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if _, ok := a.requireAdmin(w, r, logger); !ok {
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if !a.peopleBody(w, r, &body) {
		return
	}
	email, ok := normaliseEmail(body.Email)
	if !ok {
		writeAuthError(w, http.StatusBadRequest, "invalid_input", "enter a valid email address")
		return
	}
	temp, err := generateTempPassword()
	if err != nil {
		logger.ErrorContext(r.Context(), "could not generate a temporary password",
			slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())))
		writeAuthError(w, http.StatusInternalServerError, "unavailable", "could not reset the password: try again")
		return
	}
	if cerr := a.adminCall(r.Context(), "AdminSetUserPassword", map[string]any{
		"UserPoolId": a.poolID, "Username": email, "Password": temp, "Permanent": false,
	}, nil); cerr != nil {
		a.writePeopleError(w, r, logger, "reset-password", cerr)
		return
	}
	writeTempPassword(w, http.StatusOK, map[string]any{"email": email, "temporary_password": temp})
}

// peopleBody checks the content type and decodes the JSON body of a People write.
func (a *cognitoAuth) peopleBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if !isJSONRequest(r) {
		writeAuthError(w, http.StatusUnsupportedMediaType, "invalid_input", "send application/json")
		return false
	}
	return decodeBody(w, r, v)
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodyBytes)).Decode(v); err != nil && !errors.Is(err, io.EOF) {
		writeAuthError(w, http.StatusBadRequest, "invalid_input", "invalid json")
		return false
	}
	return true
}

// writeTempPassword answers with a one-time secret: never cached, by the browser or anything
// between.
func writeTempPassword(w http.ResponseWriter, status int, v map[string]any) {
	w.Header().Set("Cache-Control", "no-store")
	apikit.WriteJSON(w, status, v)
}

// writePeopleError turns a failed administrator call into an answer the screen can act on. What
// it logs is the operation and Cognito's error name, never a request body.
func (a *cognitoAuth) writePeopleError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, action string, e *cognitoError) {
	switch e.kind {
	case "UserNotFoundException":
		writeAuthError(w, http.StatusNotFound, "not_found", "there is no one with that email address")
	case "UsernameExistsException":
		writeAuthError(w, http.StatusConflict, "exists", "an account with that email already exists")
	case "InvalidParameterException":
		writeAuthError(w, http.StatusBadRequest, "invalid_input", "check the details and try again")
	case "LimitExceededException", "TooManyRequestsException":
		writeAuthError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts: wait a while and try again")
	default:
		logger.ErrorContext(r.Context(), "people call failed",
			slog.String("correlation_id", apikit.CorrelationIDFromContext(r.Context())),
			slog.String("action", action),
			slog.String("error", e.Error()))
		writeAuthError(w, http.StatusBadGateway, "unavailable", "could not reach the user pool: try again shortly")
	}
}

// normaliseEmail lowercases and checks an address the way the sign-in calls do, so the account
// the administrator creates is the one the person signs in to.
func normaliseEmail(raw string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" || len(s) > 254 || strings.ContainsAny(s, " \t\r\n<>,;\"") {
		return "", false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s || !strings.Contains(s[strings.LastIndex(s, "@"):], ".") {
		return "", false
	}
	return s, true
}

// awsCredentialsFromEnv reads the execution-role credentials the Lambda runtime provides.
func awsCredentialsFromEnv() awsCredentials {
	return awsCredentials{
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
	}
}

// The character classes of a temporary password. The pool's policy wants 12 or more characters
// with a lowercase letter, an uppercase letter and a number; the symbol is extra headroom for a
// stricter policy. No character needs escaping in a URL, a shell or a chat message.
const (
	tempLower   = "abcdefghijkmnopqrstuvwxyz" // no l
	tempUpper   = "ABCDEFGHJKLMNPQRSTUVWXYZ"  // no I, O
	tempDigits  = "23456789"                  // no 0, 1
	tempSymbols = "-_!*"
	tempLength  = 20
)

// generateTempPassword returns a random temporary password with at least one character of every
// class, from crypto/rand. It is never logged.
func generateTempPassword() (string, error) {
	all := tempLower + tempUpper + tempDigits + tempSymbols
	out := make([]byte, 0, tempLength)
	for _, class := range []string{tempLower, tempUpper, tempDigits, tempSymbols} {
		c, err := randomChar(class)
		if err != nil {
			return "", err
		}
		out = append(out, c)
	}
	for len(out) < tempLength {
		c, err := randomChar(all)
		if err != nil {
			return "", err
		}
		out = append(out, c)
	}
	// Fisher-Yates, so the guaranteed characters are not always first.
	for i := len(out) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		out[i], out[j.Int64()] = out[j.Int64()], out[i]
	}
	return string(out), nil
}

func randomChar(set string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
	if err != nil {
		return 0, err
	}
	return set[n.Int64()], nil
}
