package albitorsvc

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// The gateway's refusal codes the client branches on. The
// wire body is {"error": code}, plus {"aws_code", "request_id"} for an upstream
// failure.
const (
	CodeCreditsExhausted = "credits_exhausted"
	CodeCapReached       = "cap_reached"
)

// Error is a refusal or failure the gateway answered with. Its text is the
// status and code only: the gateway never sends AWS's message, and neither does
// this client.
type Error struct {
	Status int
	Code   string
	// RetryAfter is the gateway's Retry-After, zero when it sent none.
	RetryAfter time.Duration
	// AWSCode and RequestID are set for an upstream failure.
	AWSCode   string
	RequestID string
}

func (e *Error) Error() string {
	if e.AWSCode != "" {
		return fmt.Sprintf("albitorsvc: %d %s (%s)", e.Status, e.Code, e.AWSCode)
	}
	return fmt.Sprintf("albitorsvc: %d %s", e.Status, e.Code)
}

// Sentinels for errors.Is. A sentinel with a Code matches that status and code;
// one without matches the status alone.
var (
	// ErrCreditsExhausted is 402 credits_exhausted: the organisation has no
	// credits left, so the feature is unavailable until it tops up.
	ErrCreditsExhausted = &Error{Status: http.StatusPaymentRequired, Code: CodeCreditsExhausted}
	// ErrCapReached is 402 cap_reached: the app's daily cap is spent, so the
	// feature is unavailable until the cap resets or is raised.
	ErrCapReached = &Error{Status: http.StatusPaymentRequired, Code: CodeCapReached}
	// ErrForbidden is any 403 (not_declared, service_disabled, app_disabled,
	// app_deleted, not_available_in_residency): the app may not make this call.
	ErrForbidden = &Error{Status: http.StatusForbidden}
	// ErrRateLimited is 429, returned once the client has retried and given up.
	ErrRateLimited = &Error{Status: http.StatusTooManyRequests}
)

// Is lets errors.Is match a sentinel.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Status == t.Status && (t.Code == "" || e.Code == t.Code)
}

// Unavailable reports that the feature cannot run for this app right now and the
// app should say so and carry on: 402 (no credits, or the daily cap) or 403 (not
// declared, switched off). It is not a fault to retry. Show the end user "this
// feature is unavailable", never the code.
func Unavailable(err error) bool {
	return errors.Is(err, ErrCreditsExhausted) || errors.Is(err, ErrCapReached) || errors.Is(err, ErrForbidden)
}

// ErrNotConfigured is returned by Open when ALBITOR_SERVICES_URL is unset: the
// manifest declares no services.callable, or this is a local run.
var ErrNotConfigured = errors.New("albitorsvc: ALBITOR_SERVICES_URL is not set; declare services.callable in albitor.yaml")
