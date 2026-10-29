package albitorsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// URLEnv is set by the platform when the manifest declares services.callable.
	URLEnv = "ALBITOR_SERVICES_URL"

	// maxResponse bounds what is read from the gateway; its own ceiling is lower.
	maxResponse = 8 << 20
	// maxErrorBody bounds an error body: {"error": code} and little else.
	maxErrorBody = 4 << 10
)

var (
	servicePattern   = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)
	operationPattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]{1,63}$`)
	// The gateway's own rule for an Idempotency-Key.
	idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
	regionInHost          = regexp.MustCompile(`\.lambda-url\.([a-z0-9-]+)\.on\.aws$`)
)

// Config builds a Client. Zero values take the defaults below.
type Config struct {
	// BaseURL is the gateway's function URL (ALBITOR_SERVICES_URL).
	BaseURL string
	// Region is the signing Region. Empty takes it from the function URL's host.
	Region string
	// Credentials are the execution role's, as Lambda puts them in the environment.
	Credentials Credentials
	// HTTP is the transport; nil is a client with a 60 second timeout.
	HTTP *http.Client
	// MaxAttempts bounds tries for a 429, the first included. Default 4.
	MaxAttempts int
	// BaseDelay and MaxDelay shape the jittered backoff. Defaults 250ms and 5s.
	BaseDelay, MaxDelay time.Duration

	// Now and Sleep are test seams.
	Now   func() time.Time
	Sleep func(ctx context.Context, d time.Duration) error
}

// Client calls the gateway. It is safe for concurrent use.
type Client struct {
	base   *url.URL
	region string
	creds  Credentials
	http   *http.Client
	cfg    Config
}

// Open builds a Client from the environment, read through getenv (os.Getenv in
// main). It returns ErrNotConfigured when ALBITOR_SERVICES_URL is unset, so the
// app can run without the feature locally and in tests.
func Open(getenv func(string) string) (*Client, error) {
	env := func(k string) string { return strings.TrimSpace(getenv(k)) }
	base := env(URLEnv)
	if base == "" {
		return nil, ErrNotConfigured
	}
	return New(Config{
		BaseURL: base,
		Region:  env("AWS_REGION"),
		Credentials: Credentials{
			AccessKeyID:     env("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: env("AWS_SECRET_ACCESS_KEY"),
			SessionToken:    env("AWS_SESSION_TOKEN"),
		},
	})
}

// New validates cfg and builds a Client.
func New(cfg Config) (*Client, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && !isLoopback(u)) {
		return nil, errors.New("albitorsvc: the gateway URL must be https")
	}
	if u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return nil, errors.New("albitorsvc: the gateway URL carries a query, fragment or user")
	}
	region := cfg.Region
	if region == "" {
		if m := regionInHost.FindStringSubmatch(u.Hostname()); m != nil {
			region = m[1]
		}
	}
	if region == "" {
		return nil, errors.New("albitorsvc: no signing Region: set AWS_REGION")
	}
	if cfg.Credentials.AccessKeyID == "" || cfg.Credentials.SecretAccessKey == "" {
		return nil, errors.New("albitorsvc: no credentials: the function's role credentials are not in the environment")
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 4
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 250 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 5 * time.Second
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Sleep == nil {
		cfg.Sleep = sleep
	}
	hc := &http.Client{Timeout: 60 * time.Second}
	if cfg.HTTP != nil {
		copied := *cfg.HTTP
		hc = &copied
	}
	// The signature binds the host and path, so a redirect is never followed.
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{base: u, region: region, creds: cfg.Credentials, http: hc, cfg: cfg}, nil
}

func isLoopback(u *url.URL) bool {
	h := u.Hostname()
	return u.Scheme == "http" && (h == "127.0.0.1" || h == "localhost" || h == "::1")
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// CallOption adjusts one call.
type CallOption func(*callOptions)

type callOptions struct{ idempotencyKey string }

// WithIdempotencyKey sets the Idempotency-Key header. The gateway requires one on
// operations that start work or bill by the call (Bedrock, job starts), so a
// replay of a signed request does not charge twice. Use a fresh random value per
// logical call, and reuse it only to repeat that same call.
func WithIdempotencyKey(key string) CallOption {
	return func(o *callOptions) { o.idempotencyKey = key }
}

// Call sends req (any value that marshals to the AWS request JSON) to
// POST /v1/<service>/<operation> and decodes the response into resp (nil to
// discard it). A non-2xx answer is an *Error: test it with errors.Is against
// ErrCreditsExhausted, ErrCapReached, ErrForbidden or ErrRateLimited, or with
// Unavailable.
//
// A 429 is retried with full-jitter exponential backoff, never sooner than its
// Retry-After, up to MaxAttempts, because the gateway refuses it before the call
// runs. A Retry-After longer than MaxDelay is not waited for: Call returns the
// 429 (ErrRateLimited) at once. Nothing else is retried: a 5xx may have run, and
// a 402 or 403 will not change by asking again. Retries spend the app's own rate
// budget, so do not wrap Call in a retry loop of the app's own.
//
// A key given to WithIdempotencyKey must match ^[A-Za-z0-9_-]{8,64}$, the
// gateway's rule; Call refuses any other before sending.
func (c *Client) Call(ctx context.Context, service, operation string, req, resp any, opts ...CallOption) error {
	if !servicePattern.MatchString(service) || !operationPattern.MatchString(operation) {
		return fmt.Errorf("albitorsvc: %q/%q is not a service and operation name", service, operation)
	}
	var o callOptions
	for _, f := range opts {
		f(&o)
	}
	if o.idempotencyKey != "" && !idempotencyKeyPattern.MatchString(o.idempotencyKey) {
		return errors.New("albitorsvc: the idempotency key must match [A-Za-z0-9_-]{8,64}")
	}
	body := []byte("{}")
	if req != nil {
		var err error
		if body, err = json.Marshal(req); err != nil {
			return fmt.Errorf("albitorsvc: encode request: %w", err)
		}
	}
	endpoint := c.base.JoinPath("v1", service, operation).String()

	for attempt := 1; ; attempt++ {
		out, err := c.once(ctx, endpoint, body, o)
		if err == nil {
			if resp == nil {
				return nil
			}
			if err := json.Unmarshal(out, resp); err != nil {
				return fmt.Errorf("albitorsvc: decode response: %w", err)
			}
			return nil
		}
		var ge *Error
		if !errors.As(err, &ge) || ge.Status != http.StatusTooManyRequests || attempt >= c.cfg.MaxAttempts {
			return err
		}
		// A wait longer than the cap is the gateway saying "not soon": give up now
		// rather than retry early and spend the app's rate budget for nothing.
		if ge.RetryAfter > c.cfg.MaxDelay {
			return err
		}
		if err := c.cfg.Sleep(ctx, c.backoff(attempt, ge.RetryAfter)); err != nil {
			return err
		}
	}
}

// backoff is full jitter over a capped exponential, and never less than the
// gateway's Retry-After, which is itself capped so a hostile value cannot park a
// request.
func (c *Client) backoff(attempt int, retryAfter time.Duration) time.Duration {
	ceiling := min(c.cfg.BaseDelay<<(attempt-1), c.cfg.MaxDelay)
	d := time.Duration(rand.Int64N(int64(ceiling) + 1))
	return max(d, min(retryAfter, c.cfg.MaxDelay))
}

func (c *Client) once(ctx context.Context, endpoint string, body []byte, o callOptions) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", o.idempotencyKey)
	}
	signRequest(req, body, c.creds, c.region, "lambda", c.cfg.Now())
	res, err := c.http.Do(req)
	if err != nil {
		// Cancellation stays recognisable; any other failure is reduced to a
		// fixed text, since a transport error can echo the URL.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("albitorsvc: the request failed to send")
	}
	defer res.Body.Close()
	if res.StatusCode/100 == 2 {
		out, err := io.ReadAll(io.LimitReader(res.Body, maxResponse+1))
		if err != nil {
			return nil, errors.New("albitorsvc: the response could not be read")
		}
		if len(out) > maxResponse {
			return nil, errors.New("albitorsvc: response too large")
		}
		return out, nil
	}
	raw, _ := io.ReadAll(io.LimitReader(res.Body, maxErrorBody))
	return nil, parseError(res.StatusCode, res.Header.Get("Retry-After"), raw)
}

func parseError(status int, retryAfter string, raw []byte) *Error {
	e := &Error{Status: status, Code: "unknown"}
	var b struct {
		Error     string `json:"error"`
		AWSCode   string `json:"aws_code"`
		RequestID string `json:"request_id"`
	}
	if json.Unmarshal(raw, &b) == nil && b.Error != "" {
		e.Code, e.AWSCode, e.RequestID = b.Error, b.AWSCode, b.RequestID
	}
	if n, err := strconv.Atoi(retryAfter); err == nil && n > 0 {
		e.RetryAfter = time.Duration(n) * time.Second
	}
	return e
}
