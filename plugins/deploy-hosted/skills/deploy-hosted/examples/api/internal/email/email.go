// Package email is how a hosted app sends the mail it declared: a template by id, to one
// address, with typed values, through the service gateway's email service.
//
// The app never holds a sender, a subject, a body or a link host. The wording of every
// template is the owner's, confirmed in the platform before the build, and the gateway renders
// it. Send names a template id that appears in albitor.yaml under
// services.callable[service: email].templates, the address, and exactly the values that template
// declares. The gateway refuses anything else (400 bad_request, 400 invalid_variable, 403
// not_declared), and Send checks the same limits first so a mistake fails in a test, not in
// production.
//
// A send that is not delivered is not an error. The gateway answers suppressed when the
// recipient is not to be mailed (their preference is off, the template is unconfirmed, or a
// members-only template met a non-member), and Send returns that as a Disposition for the
// caller to log. Never retry around it.
package email

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"webapp/internal/gateway"
)

const (
	// Service is the gateway's name for the email service.
	Service = "email"

	// MaxVars and MaxVarRunes are the gateway's limits on one send's values.
	MaxVars     = 12
	MaxVarRunes = 500
	// MaxAddressBytes is the gateway's limit on To.
	MaxAddressBytes = 254
)

var (
	templatePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)
	varNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
	numberPattern   = regexp.MustCompile(`^[0-9]{1,18}(\.[0-9]{1,6})?$`)
)

// Disposition is what the gateway did with a send.
type Disposition string

const (
	// Sent: the recipient is a member and the mail was accepted.
	Sent Disposition = "sent"
	// Invited: the recipient is not a member, so the platform's fixed invitation went instead.
	Invited Disposition = "invited"
	// Suppressed: nothing was sent, and that is correct. A recipient who switched an optional
	// mail off, who never agreed to a consent mail, a template the owner has not confirmed, and a
	// members-only template for a non-member all read this way. Log it and carry on.
	Suppressed Disposition = "suppressed"
)

// Result is the gateway's answer to a send that was not refused.
type Result struct {
	MessageID   string
	Disposition Disposition
}

// Caller is the one method of the gateway client the package uses, so tests fake it.
type Caller interface {
	Call(ctx context.Context, service, operation string, req, resp any, opts ...gateway.CallOption) error
}

// Sender sends the app's declared templates.
type Sender struct {
	gw       Caller
	declared []string
	log      *slog.Logger
}

// New builds a Sender. declared is the template ids the manifest lists under
// services.callable[service: email].templates, kept in one place in main so the manifest and the
// code cannot drift: a template id outside it is a bug, refused before any call.
func New(gw Caller, declared []string, log *slog.Logger) (*Sender, error) {
	if gw == nil {
		return nil, errors.New("email: no gateway client")
	}
	if len(declared) == 0 || len(declared) > 12 {
		return nil, errors.New("email: declare one to twelve templates")
	}
	for _, id := range declared {
		if !templatePattern.MatchString(id) {
			return nil, fmt.Errorf("email: %q is not a template id", id)
		}
	}
	if log == nil {
		log = slog.Default()
	}
	return &Sender{gw: gw, declared: slices.Clone(declared), log: log}, nil
}

// Var is one typed value of a template. Build it with Text, Number or Link, never by hand; a bad
// value is held and reported by Send.
type Var struct {
	value string
	err   error
}

// Vars are a template's values by placeholder name. The names are exactly the template's:
// a missing, extra or wrongly typed one is refused.
type Vars map[string]Var

// Text is a short single line with no link in it (a name, an order number, a date as words).
func Text(s string) Var {
	switch {
	case strings.TrimSpace(s) == "":
		return Var{err: errors.New("text is empty")}
	case strings.ContainsAny(s, "\r\n\t"):
		return Var{err: errors.New("text spans lines")}
	case !utf8.ValidString(s):
		return Var{err: errors.New("text is not valid UTF-8")}
	case utf8.RuneCountInString(s) > MaxVarRunes:
		return Var{err: fmt.Errorf("text is over %d characters", MaxVarRunes)}
	case strings.Contains(s, "://") || strings.Contains(strings.ToLower(s), "www."):
		return Var{err: errors.New("text holds a link: use Link")}
	}
	return Var{value: s}
}

// Number is a non-negative integer or plain decimal, from a value the app computed.
func Number(s string) Var {
	if !numberPattern.MatchString(s) {
		return Var{err: errors.New("number is not a non-negative integer or plain decimal")}
	}
	return Var{value: s}
}

// Link is an https link on the app's own host: origin is "https://<host>" and path starts with
// "/". The gateway holds every link to the host the platform knows for the app and refuses any
// other (400 invalid_variable), so a wrong origin fails here, not by reaching a stranger.
func Link(origin, path string) Var {
	if !strings.HasPrefix(origin, "https://") || strings.ContainsAny(origin[len("https://"):], "/?#@: ") {
		return Var{err: errors.New("link origin must be https://<host>")}
	}
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.ContainsAny(path, " \r\n\t") {
		return Var{err: errors.New("link path must start with a single /")}
	}
	s := origin + path
	if utf8.RuneCountInString(s) > MaxVarRunes {
		return Var{err: fmt.Errorf("link is over %d characters", MaxVarRunes)}
	}
	return Var{value: s}
}

// Option adjusts one send.
type Option func(*options)

type options struct{ key string }

// ForEvent names the business event the mail is for, such as "order-1042-receipt". The
// idempotency key is derived from the event, the template and the address (case-insensitive), so
// one event can mail several people or several templates, and a repeat send of the same one
// returns the first answer and sends nothing: a handler that runs twice does not mail twice.
// Without it every send is its own event, which is right for a mail that may legitimately repeat.
func ForEvent(event string) Option { return func(o *options) { o.key = event } }

// Send sends a declared template to one address. A refusal is an error; a send the gateway
// chose not to deliver is a Result with Disposition Suppressed.
//
// Send never retries and never waits: the gateway client already retried a 429, and a mail the
// app cannot send is the app's to log, not to loop on.
func (s *Sender) Send(ctx context.Context, template, to string, vars Vars, opts ...Option) (Result, error) {
	var o options
	for _, f := range opts {
		f(&o)
	}
	if !templatePattern.MatchString(template) {
		return Result{}, fmt.Errorf("email: %q is not a template id", template)
	}
	if !slices.Contains(s.declared, template) {
		return Result{}, fmt.Errorf("email: template %q is not declared in albitor.yaml services.callable email templates", template)
	}
	if err := checkAddress(to); err != nil {
		return Result{}, err
	}
	values, err := checkVars(vars)
	if err != nil {
		return Result{}, fmt.Errorf("email: template %q: %w", template, err)
	}
	key := o.key
	if key == "" {
		if key, err = randomKey(); err != nil {
			return Result{}, err
		}
	} else {
		key = eventKey(key, template, to)
	}
	req := struct {
		TemplateID     string            `json:"TemplateId"`
		To             string            `json:"To"`
		Variables      map[string]string `json:"Variables,omitempty"`
		IdempotencyKey string            `json:"IdempotencyKey"`
	}{template, to, values, key}
	var resp struct {
		MessageID   string `json:"MessageId"`
		Disposition string `json:"Disposition"`
	}
	if err := s.gw.Call(ctx, Service, "SendTemplatedEmail", req, &resp); err != nil {
		return Result{}, err
	}
	res := Result{MessageID: resp.MessageID, Disposition: Disposition(resp.Disposition)}
	if res.Disposition == Suppressed {
		// Not an error and not a retry: the recipient's choice, an unconfirmed template or a
		// members-only mail is a normal answer. No address in the log.
		s.log.InfoContext(ctx, "email: not sent", "template", template, "disposition", string(res.Disposition))
	}
	return res, nil
}

// Outcome is what a caller does with a failed send, so the handler's branch is one switch and
// no failure is retried.
type Outcome int

const (
	// OutcomeFault is a bug or an outage: log it with the correlation id and answer an opaque
	// 500 (or carry on, for a mail that is not the point of the request). Includes 400, which is a
	// build bug in the request shape and is never retried.
	OutcomeFault Outcome = iota
	// OutcomeSkip is a refusal that means "this mail is not going": the template is unknown to
	// the cell or not declared (unknown_template, not_declared), the address is on the platform's
	// never-email list (never_email), the invitation window or the daily cap is spent
	// (invite_window, daily_cap), the mail tenant is paused (tenant_paused), or the app has no
	// credits. Log once and carry on; do not retry and do not queue it for later.
	OutcomeSkip
)

// Classify says what to do with an error from Send. The codes are the gateway's own
// : none is ever shown to a person.
func Classify(err error) Outcome {
	var ge *gateway.Error
	if !errors.As(err, &ge) {
		return OutcomeFault
	}
	switch {
	case gateway.Unavailable(err):
		return OutcomeSkip
	case ge.Status == http.StatusNotFound && ge.Code == "unknown_template":
		return OutcomeSkip
	case ge.Status == http.StatusConflict && (ge.Code == "never_email" || ge.Code == "invite_window"):
		return OutcomeSkip
	case ge.Status == http.StatusTooManyRequests:
		return OutcomeSkip
	}
	return OutcomeFault
}

func checkAddress(to string) error {
	switch {
	case to == "" || len(to) > MaxAddressBytes:
		return errors.New("email: the address is empty or over 254 bytes")
	case strings.ContainsAny(to, " <>\",;()\r\n\t") || strings.Count(to, "@") != 1:
		return errors.New("email: one bare address only, no name, comment or list")
	}
	return nil
}

func checkVars(vars Vars) (map[string]string, error) {
	if len(vars) > MaxVars {
		return nil, fmt.Errorf("more than %d values", MaxVars)
	}
	out := make(map[string]string, len(vars))
	for name, v := range vars {
		if !varNamePattern.MatchString(name) {
			return nil, fmt.Errorf("%q is not a placeholder name", name)
		}
		if v.err != nil {
			return nil, fmt.Errorf("value %q: %w", name, v.err)
		}
		if v.value == "" {
			return nil, fmt.Errorf("value %q is unset: build it with Text, Number or Link", name)
		}
		out[name] = v.value
	}
	return out, nil
}

// eventKey is a deterministic gateway idempotency key for one event, template and recipient: a
// hex sha256 prefix, so it always matches [A-Za-z0-9._:-]{8,64} whatever the event text.
func eventKey(event, template, to string) string {
	h := sha256.Sum256([]byte(event + "\x00" + template + "\x00" + strings.ToLower(to)))
	return hex.EncodeToString(h[:])[:40]
}

func randomKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("email: no random key: %w", err)
	}
	return hex.EncodeToString(b), nil
}
