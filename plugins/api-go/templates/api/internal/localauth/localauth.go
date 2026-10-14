// Package localauth is a local stand-in for the user pool: an issuer that runs
// in the API's own process during a local run, so specs sign in as named test
// identities with no sign-up, no e-mail and no call to AWS.
//
// It mints ID and access tokens shaped like Cognito's for a fixed set of
// identities, and serves its JWKS, under Path:
//
//	POST /api/_local/auth/token                  {"identity":"alice"}
//	GET  /api/_local/auth/identities
//	GET  /api/_local/auth/.well-known/jwks.json
//
// It only issues tokens. It does not verify them, and this package changes
// nothing about how the API treats a request: an app with no sign-in still
// answers every request as its anonymous owner. An app whose sign-in verifies
// tokens trusts this issuer only in a local run, under the same gates as Open.
//
// THE CONTRACT A VERIFIER READS. Issuer is http://localhost:<PORT>/api/_local/auth
// (PORT defaults to 8080), its keys are at Issuer + "/.well-known/jwks.json",
// and its tokens are for ClientID.
//
// This is platform code, maintained with the local run it serves; extend the
// app around it rather than editing it.
package localauth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	// EnabledEnv starts the issuer. Only "1" does.
	EnabledEnv = "ALBITOR_LOCAL_AUTH"
	// LocalModeEnv is the local run's own switch; the issuer needs it.
	LocalModeEnv = "ALBITOR_LOCAL"
	// Path is where the issuer's routes are served, and the path of its
	// issuer URL.
	Path = "/api/_local/auth"
	// ClientID is the app client the issuer's tokens are for.
	ClientID = "local-test-client"
	// TokenTTL is how long a token lives, as Cognito's default.
	TokenTTL = time.Hour
	// keyBits is the size of the per-process signing key.
	keyBits = 2048
)

// deployedMarkers are set by the Lambda runtime itself, so configuration
// cannot unset them.
var deployedMarkers = []string{"AWS_LAMBDA_FUNCTION_NAME"}

// realPoolSettings name a real user pool. Any of them set means the app
// trusts that pool, and the local issuer must not run beside it.
var realPoolSettings = []string{"AUTH_ISSUER", "AUTH_USER_POOL_ID", "AUTH_CLIENT_ID"}

var (
	// ErrNotLocal refuses the issuer outside a local run.
	ErrNotLocal = errors.New(EnabledEnv + "=1 needs " + LocalModeEnv + "=1")
	// ErrDeployed refuses the issuer where a deployed environment marker is
	// present.
	ErrDeployed = errors.New(EnabledEnv + "=1 is refused in a deployed environment")
	// ErrRealPool refuses the issuer when a real user pool is configured.
	ErrRealPool = errors.New(EnabledEnv + "=1 is refused when a user pool is configured")
)

// IssuerURL is the issuer URL a local run on port serves.
func IssuerURL(port string) string {
	if strings.TrimSpace(port) == "" {
		port = "8080"
	}
	return "http://localhost:" + strings.TrimSpace(port) + Path
}

// Identity is one named test identity.
type Identity struct {
	Name    string   `json:"name"`
	Subject string   `json:"sub"`
	Email   string   `json:"email"`
	Groups  []string `json:"groups"`
}

// identities are the identities a spec signs in as: two users, to prove one
// cannot see the other's records, and an admin. Their subjects are fixed, so a
// record written in one run belongs to the same user in the next.
var identities = map[string]Identity{
	"alice": {Name: "alice", Subject: "00000000-0000-4000-8000-00000000a11c", Email: "alice@example.test", Groups: []string{}},
	"bob":   {Name: "bob", Subject: "00000000-0000-4000-8000-000000000b0b", Email: "bob@example.test", Groups: []string{}},
	"admin": {Name: "admin", Subject: "00000000-0000-4000-8000-00000000ad31", Email: "admin@example.test", Groups: []string{"admin"}},
}

// Issuer mints tokens for the test identities. Its signing key is generated
// when it starts and never leaves the process, so no token it mints verifies
// anywhere else, or after a restart.
type Issuer struct {
	issuer string
	kid    string
	key    *rsa.PrivateKey
	jwks   []byte
	now    func() time.Time
}

// Open starts the issuer when the environment, read through getenv, asks for
// it, and returns nil when it does not. Each guard holds on its own:
//
//   - only ALBITOR_LOCAL_AUTH=1 starts it;
//   - it needs ALBITOR_LOCAL=1, the local run's own switch, which the platform
//     never sets and an app cannot set as a config key;
//   - it refuses where the Lambda runtime's own marker is set;
//   - it refuses beside a configured user pool (AUTH_ISSUER,
//     AUTH_USER_POOL_ID or AUTH_CLIENT_ID).
func Open(getenv func(string) string) (*Issuer, error) {
	if strings.TrimSpace(getenv(EnabledEnv)) != "1" {
		return nil, nil
	}
	for _, name := range realPoolSettings {
		if strings.TrimSpace(getenv(name)) != "" {
			return nil, fmt.Errorf("%w (%s is set)", ErrRealPool, name)
		}
	}
	if strings.TrimSpace(getenv(LocalModeEnv)) != "1" {
		return nil, ErrNotLocal
	}
	for _, marker := range deployedMarkers {
		if getenv(marker) != "" {
			return nil, fmt.Errorf("%w (%s is set)", ErrDeployed, marker)
		}
	}
	return newIssuer(IssuerURL(getenv("PORT")))
}

func newIssuer(issuer string) (*Issuer, error) {
	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, fmt.Errorf("generate the local issuer's key: %w", err)
	}
	var kidBytes [8]byte
	_, _ = rand.Read(kidBytes[:])
	l := &Issuer{issuer: issuer, kid: "local-" + hex.EncodeToString(kidBytes[:]), key: key, now: time.Now}
	l.jwks, err = json.Marshal(map[string][]map[string]string{"keys": {{
		"kty": "RSA",
		"kid": l.kid,
		"use": "sig",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}})
	if err != nil {
		return nil, err
	}
	return l, nil
}

// URL is the `iss` the issuer's tokens carry.
func (l *Issuer) URL() string { return l.issuer }

// Tokens is what the token route answers.
type Tokens struct {
	IDToken     string   `json:"id_token"`
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
	Identity    Identity `json:"identity"`
}

// Mint returns an ID token and an access token for the named identity, with
// the claims Cognito's carry.
func (l *Issuer) Mint(name string) (Tokens, error) {
	id, ok := identities[name]
	if !ok {
		return Tokens{}, fmt.Errorf("no local identity %q", name)
	}
	now := l.now()
	exp := now.Add(TokenTTL)
	idToken, err := l.sign(map[string]any{
		"iss":              l.issuer,
		"sub":              id.Subject,
		"aud":              ClientID,
		"token_use":        "id",
		"email":            id.Email,
		"email_verified":   true,
		"cognito:username": id.Name,
		"cognito:groups":   id.Groups,
		"auth_time":        now.Unix(),
		"iat":              now.Unix(),
		"exp":              exp.Unix(),
	})
	if err != nil {
		return Tokens{}, err
	}
	accessToken, err := l.sign(map[string]any{
		"iss":            l.issuer,
		"sub":            id.Subject,
		"client_id":      ClientID,
		"token_use":      "access",
		"scope":          "aws.cognito.signin.user.admin",
		"username":       id.Name,
		"cognito:groups": id.Groups,
		"auth_time":      now.Unix(),
		"iat":            now.Unix(),
		"exp":            exp.Unix(),
	})
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{
		IDToken:     idToken,
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(TokenTTL / time.Second),
		Identity:    id,
	}, nil
}

func (l *Issuer) sign(claims map[string]any) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": l.kid, "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, l.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// Wrap serves the issuer's routes in front of app. Everything under Path is
// the issuer's, a 404 where it has no route; everything else is app's.
func (l *Issuer) Wrap(app http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == Path || strings.HasPrefix(r.URL.Path, Path+"/") {
			l.ServeHTTP(w, r)
			return
		}
		app.ServeHTTP(w, r)
	})
}

// ServeHTTP answers the issuer's routes.
func (l *Issuer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == Path+"/.well-known/jwks.json" && r.Method == http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(l.jwks)
	case r.URL.Path == Path+"/identities" && r.Method == http.MethodGet:
		names := make([]string, 0, len(identities))
		for name := range identities {
			names = append(names, name)
		}
		sort.Strings(names)
		out := make([]Identity, 0, len(names))
		for _, name := range names {
			out = append(out, identities[name])
		}
		writeJSON(w, http.StatusOK, map[string]any{"issuer": l.issuer, "client_id": ClientID, "identities": out})
	case r.URL.Path == Path+"/token" && r.Method == http.MethodPost:
		var body struct {
			Identity string `json:"identity"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		tokens, err := l.Mint(body.Identity)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown identity"})
			return
		}
		writeJSON(w, http.StatusOK, tokens)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
