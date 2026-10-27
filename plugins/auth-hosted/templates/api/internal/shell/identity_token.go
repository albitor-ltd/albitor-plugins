package shell

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Verifying a Cognito token, with the standard library only.
//
// The issuer, the client and the keys' address come from the environment the hosted deploy hands
// the function (AUTH_ISSUER, AUTH_CLIENT_ID, AUTH_JWKS_URL), not from a pool id.
//
// Cognito signs every token RS256 with a key from the pool's JWKS. The check is
// the one AWS documents: the signature against the key the token's `kid` names,
// then `iss` (this pool), `token_use` and the client (`aud` on an ID token,
// `client_id` on an access token), then expiry. Anything that fails is refused,
// and the caller is told nothing about why.
//
// It is written out here rather than taken from a JWT library so the API's
// go.mod needs nothing new: RS256 is one hash and one RSA check, and the only
// algorithm accepted. A token naming any other `alg` (`none` included) is
// refused before its claims are read.

var (
	// errTokenInvalid is any token that is not a valid token for this pool and
	// client.
	errTokenInvalid = errors.New("token is not valid")
	// errTokenExpired is a token that WAS valid: correctly signed for this pool
	// and client, and now past its expiry. Only this one may be refreshed.
	errTokenExpired = errors.New("token has expired")
	// errVerifierUnavailable is a token that could not be checked because the
	// pool's keys could not be fetched. It is the platform's failure, not the
	// caller's, so it does not end their session.
	errVerifierUnavailable = errors.New("token keys are unavailable")
	// errUnknownKey is a token whose kid the pool's keys do not hold. It is refused like an invalid
	// token, but it is not proof the token is bad: the pool may have rotated its keys since the
	// last fetch, so a session cookie carrying one is not cleared.
	errUnknownKey = errors.New("token key is unknown")
)

const (
	// tokenClockSkew is how far a token's iat may sit in the future.
	tokenClockSkew = 2 * time.Minute
	// jwksRefetchInterval bounds how often an unknown kid refetches the JWKS,
	// so a flood of forged kids cannot turn into a flood of fetches.
	jwksRefetchInterval = 5 * time.Minute
	// jwksFailureTTL is how long a failed fetch is remembered, so a pool that is down is asked
	// once in a while and not once per request.
	jwksFailureTTL = 10 * time.Second
	maxJWKSBytes   = 1 << 20
)

// cognitoVerifier checks tokens for one pool and client. It is safe for
// concurrent use.
type cognitoVerifier struct {
	issuer   string
	clientID string
	jwksURL  string
	client   *http.Client
	now      func() time.Time

	fetchMu   sync.Mutex // serialises fetches; never held with mu across the network
	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	failedAt  time.Time
}

func newCognitoVerifier(issuer, clientID, jwksURL string, client *http.Client) *cognitoVerifier {
	return &cognitoVerifier{
		issuer:   issuer,
		clientID: clientID,
		jwksURL:  jwksURL,
		client:   client,
		now:      time.Now,
		keys:     map[string]*rsa.PublicKey{},
	}
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type jwtClaims struct {
	Iss           string   `json:"iss"`
	Sub           string   `json:"sub"`
	Aud           string   `json:"aud"`
	ClientID      string   `json:"client_id"`
	TokenUse      string   `json:"token_use"`
	Email         string   `json:"email"`
	EmailVerified flexBool `json:"email_verified"`
	Scope         string   `json:"scope"`
	Groups        []string `json:"cognito:groups"`
	Exp           int64    `json:"exp"`
	Iat           int64    `json:"iat"`
}

// flexBool reads a JSON boolean, or the strings "true" and "false", which Cognito has emitted for
// email_verified. Anything else is false: an unverified address is never assumed verified.
type flexBool bool

func (b *flexBool) UnmarshalJSON(raw []byte) error {
	switch strings.Trim(string(raw), `"`) {
	case "true":
		*b = true
	default:
		*b = false
	}
	return nil
}

// hasScope reports whether a space-separated scope claim holds scope.
func (c jwtClaims) hasScope(scope string) bool {
	for _, s := range strings.Fields(c.Scope) {
		if s == scope {
			return true
		}
	}
	return false
}

// verifyClaims returns every claim of a token that passed the check, errTokenExpired for a genuine
// token past its expiry, errVerifierUnavailable when the pool's keys cannot be fetched, or
// errTokenInvalid.
func (v *cognitoVerifier) verifyClaims(ctx context.Context, raw string) (jwtClaims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return jwtClaims{}, errTokenInvalid
	}
	var header jwtHeader
	if err := decodeJWTSegment(parts[0], &header); err != nil || header.Alg != "RS256" || header.Kid == "" {
		return jwtClaims{}, errTokenInvalid
	}
	key, err := v.key(ctx, header.Kid)
	if err != nil {
		return jwtClaims{}, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtClaims{}, errTokenInvalid
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig) != nil {
		return jwtClaims{}, errTokenInvalid
	}

	var claims jwtClaims
	if err := decodeJWTSegment(parts[1], &claims); err != nil {
		return jwtClaims{}, errTokenInvalid
	}
	if claims.Iss != v.issuer || claims.Sub == "" {
		return jwtClaims{}, errTokenInvalid
	}
	switch claims.TokenUse {
	case "id":
		if claims.Aud != v.clientID {
			return jwtClaims{}, errTokenInvalid
		}
	case "access":
		if claims.ClientID != v.clientID {
			return jwtClaims{}, errTokenInvalid
		}
	default:
		return jwtClaims{}, errTokenInvalid
	}
	now := v.now()
	if claims.Iat > now.Add(tokenClockSkew).Unix() {
		return jwtClaims{}, errTokenInvalid
	}
	if now.Unix() >= claims.Exp { // a missing exp is 0: expired
		return jwtClaims{}, errTokenExpired
	}
	return claims, nil
}

func decodeJWTSegment(seg string, into any) error {
	b, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, into)
}

// key returns the pool's public key for kid, fetching the JWKS when the kid is
// unknown and the last fetch is old enough. A fetch failure is not cached.
func (v *cognitoVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if k, err, done := v.cached(kid); done {
		return k, err
	}
	// One fetch at a time, and never under mu: a slow pool must not block requests whose key is
	// already known.
	v.fetchMu.Lock()
	defer v.fetchMu.Unlock()
	if k, err, done := v.cached(kid); done { // another request fetched while this one waited
		return k, err
	}
	keys, err := v.fetchJWKS(ctx)
	v.mu.Lock()
	defer v.mu.Unlock()
	if err != nil {
		v.failedAt = v.now()
		return nil, fmt.Errorf("%w: %w", errVerifierUnavailable, err)
	}
	v.keys = keys
	v.fetchedAt = v.now()
	v.failedAt = time.Time{}
	if k, ok := v.keys[kid]; ok {
		return k, nil
	}
	return nil, errUnknownKey
}

// cached answers from what is already known: the key, an unknown kid inside the refetch interval,
// or a fetch that failed moments ago. done is false when a fetch is due.
func (v *cognitoVerifier) cached(kid string) (key *rsa.PublicKey, err error, done bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if k, ok := v.keys[kid]; ok {
		return k, nil, true
	}
	now := v.now()
	if !v.failedAt.IsZero() && now.Sub(v.failedAt) < jwksFailureTTL {
		return nil, errVerifierUnavailable, true
	}
	if !v.fetchedAt.IsZero() && now.Sub(v.fetchedAt) < jwksRefetchInterval {
		return nil, errUnknownKey, true
	}
	return nil, nil, false
}

type cognitoJWKS struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Alg string `json:"alg"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func (v *cognitoVerifier) fetchJWKS(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS answered %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJWKSBytes))
	if err != nil {
		return nil, err
	}
	var set cognitoJWKS
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" || (k.Alg != "" && k.Alg != "RS256") || k.Kid == "" {
			continue
		}
		n, errN := base64.RawURLEncoding.DecodeString(k.N)
		e, errE := base64.RawURLEncoding.DecodeString(k.E)
		if errN != nil || errE != nil || len(e) == 0 || len(e) > 4 {
			continue
		}
		exp := new(big.Int).SetBytes(e)
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exp.Int64())}
	}
	if len(keys) == 0 {
		return nil, errors.New("JWKS holds no RS256 key")
	}
	return keys, nil
}
