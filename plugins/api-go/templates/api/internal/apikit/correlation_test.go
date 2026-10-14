package apikit

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func logged(t *testing.T, h http.Handler, header string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	if header != "" {
		req.Header.Set(CorrelationIDHeader, header)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

var teapot = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })

// A bare mux echoes no correlation id, so a user-reported failure cannot be
// found in the log. Error responses carry one too: they are the ones reported.
func TestResponseCarriesCorrelationIDHeader(t *testing.T) {
	rec := logged(t, RequestLogging(teapot, quietLogger()), "")
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get(CorrelationIDHeader) == "" {
		t.Fatalf("%s header is absent", CorrelationIDHeader)
	}
}

// An inbound id is propagated, not replaced: it is what joins this service's
// logs to the caller's.
func TestInboundCorrelationIDIsPropagated(t *testing.T) {
	const inbound = "caller-supplied-0123456789"
	rec := logged(t, RequestLogging(teapot, quietLogger()), inbound)
	if got := rec.Header().Get(CorrelationIDHeader); got != inbound {
		t.Fatalf("%s = %q, want %q", CorrelationIDHeader, got, inbound)
	}
}

// An inbound id reaches a response header and a log line, so an unusable one
// is replaced rather than trusted. Header injection is the attack.
func TestUnsafeInboundCorrelationIDIsReplaced(t *testing.T) {
	cases := map[string]string{
		"header injection": "abc\r\nX-Injected: yes",
		"space":            "a b",
		"quote":            `a"b`,
		"over length":      strings.Repeat("x", maxCorrelationIDLength+1),
	}
	for name, inbound := range cases {
		t.Run(name, func(t *testing.T) {
			// httptest.NewRequest accepts a CR/LF value; the wire would not,
			// but a handler behind a proxy must not rely on that.
			rec := logged(t, RequestLogging(teapot, quietLogger()), inbound)
			got := rec.Header().Get(CorrelationIDHeader)
			if got == "" || got == inbound || !isSafeCorrelationID(got) {
				t.Fatalf("%s = %q, want a safe server-minted replacement", CorrelationIDHeader, got)
			}
		})
	}
}

// One structured line per request, with the fields an incident is rebuilt
// from, a 5xx at error level, and no query string.
func TestRequestLogLineIsStructuredAndCorrelated(t *testing.T) {
	var sink bytes.Buffer
	h := RequestLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}), slog.New(slog.NewJSONHandler(&sink, nil)))

	req := httptest.NewRequest(http.MethodGet, "/api/x?token=secret", nil)
	req.Header.Set(CorrelationIDHeader, "caller-1")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if strings.Count(strings.TrimSpace(sink.String()), "\n") != 0 {
		t.Fatalf("want exactly one line, got %q", sink.String())
	}
	var line map[string]any
	if err := json.Unmarshal(sink.Bytes(), &line); err != nil {
		t.Fatalf("log line is not JSON (%q): %v", sink.String(), err)
	}
	for field, want := range map[string]any{
		"correlation_id": "caller-1",
		"method":         http.MethodGet,
		"path":           "/api/x",
		"status":         float64(http.StatusInternalServerError),
		"level":          "ERROR",
	} {
		if line[field] != want {
			t.Fatalf("log field %q = %v, want %v", field, line[field], want)
		}
	}
	if _, ok := line["duration_ms"]; !ok {
		t.Fatalf("log line has no duration_ms: %v", line)
	}
	if strings.Contains(sink.String(), "secret") {
		t.Fatalf("the query string reached the log: %s", sink.String())
	}
}

func TestCorrelationIDIsOnTheRequestContext(t *testing.T) {
	var seen string
	h := RequestLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = CorrelationIDFromContext(r.Context())
	}), quietLogger())

	rec := logged(t, h, "")
	if seen == "" || rec.Header().Get(CorrelationIDHeader) != seen {
		t.Fatalf("context id %q does not match the echoed header %q", seen, rec.Header().Get(CorrelationIDHeader))
	}
	if got := CorrelationIDFromContext(httptest.NewRequest(http.MethodGet, "/", nil).Context()); got != "" {
		t.Fatalf("id outside a request = %q, want empty", got)
	}
}
