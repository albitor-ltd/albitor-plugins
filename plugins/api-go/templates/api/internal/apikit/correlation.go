package apikit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// CorrelationIDHeader carries the correlation id in both directions. An
// inbound value is propagated; otherwise the server mints one. Echoing it on
// the response is what makes a user-reported failure traceable: the id is
// visible in the browser's network tab and matches the log line.
const CorrelationIDHeader = "X-Correlation-Id"

// A correlation id reaches a log line and a response header, so it is
// validated before use. An unchecked header value can carry CR/LF and split
// the response header, or forge a line in a text log.
const maxCorrelationIDLength = 128

type correlationIDContextKey struct{}

// CorrelationIDFromContext returns the correlation id for the request being
// served, or "" outside a request. Every log line a handler emits should carry
// it, so one request's work can be reassembled from the log.
func CorrelationIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(correlationIDContextKey{}).(string)
	return id
}

// RequestLogging assigns or propagates a correlation id and emits one
// structured line per request.
//
// WHY THIS IS THE FIRST THING WRAPPED ROUND THE MUX. An application whose only
// log calls are in main has nothing to investigate an incident with: there is
// no record of which requests arrived, what they answered, or how long they
// took. One line per request, keyed by a correlation id, is the floor.
func RequestLogging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := resolveCorrelationID(r.Header.Get(CorrelationIDHeader))

		// Set before the handler runs: once the handler writes a body the
		// headers are already on the wire.
		w.Header().Set(CorrelationIDHeader, correlationID)

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(recorder, r.WithContext(
			context.WithValue(r.Context(), correlationIDContextKey{}, correlationID),
		))

		// A 5xx is an incident, not traffic, so it is logged at a level that
		// an alarm can filter on.
		level := slog.LevelInfo
		if recorder.status >= http.StatusInternalServerError {
			level = slog.LevelError
		}
		// The path is logged without its query string: query parameters carry
		// user input and belong in neither logs nor log-derived metrics.
		logger.LogAttrs(r.Context(), level, "http request",
			slog.String("correlation_id", correlationID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		)
	})
}

// resolveCorrelationID propagates a usable inbound id and mints one otherwise.
// A caller-supplied id is what links this service's logs to the caller's.
func resolveCorrelationID(inbound string) string {
	if id := strings.TrimSpace(inbound); isSafeCorrelationID(id) {
		return id
	}
	return newCorrelationID()
}

func isSafeCorrelationID(id string) bool {
	return IsSafeToken(id, maxCorrelationIDLength)
}

// IsSafeToken accepts a bounded, opaque, client-supplied identifier. The
// character set is deliberately narrow: these values are echoed into headers
// and log lines, where CR/LF and quoting are the attack.
func IsSafeToken(token string, maxLength int) bool {
	if token == "" || len(token) > maxLength {
		return false
	}
	for _, r := range token {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == ':':
		default:
			return false
		}
	}
	return true
}

func newCorrelationID() string {
	var buf [16]byte
	// crypto/rand.Read never returns an error on any supported platform; it
	// panics internally if the system source fails, which is the right
	// outcome for a process that cannot produce a unique id.
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}

// statusRecorder captures the status code for the access log. A handler that
// never calls WriteHeader answered 200.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(status int) {
	if s.wroteHeader {
		return
	}
	s.wroteHeader = true
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	s.wroteHeader = true
	return s.ResponseWriter.Write(b)
}
