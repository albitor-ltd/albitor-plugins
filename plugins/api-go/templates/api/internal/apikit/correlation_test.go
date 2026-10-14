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

// A panicking handler is a logged JSON 500 under the request's correlation id;
// the client sees the usual error shape and no stack.
func TestPanicIsALoggedJSON500WithNoStackForTheClient(t *testing.T) {
	var sink bytes.Buffer
	h := RequestLogging(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom: secret detail")
	}), slog.New(slog.NewJSONHandler(&sink, nil)))

	rec := logged(t, h, "caller-9")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := rec.Header().Get(CorrelationIDHeader); got != "caller-9" {
		t.Fatalf("%s = %q, want caller-9", CorrelationIDHeader, got)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"error":"internal error"}` {
		t.Fatalf("body = %s, want the JSON error shape", body)
	}
	if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "goroutine") {
		t.Fatalf("panic detail reached the client: %s", rec.Body.String())
	}

	var panicked, access map[string]any
	for _, l := range strings.Split(strings.TrimSpace(sink.String()), "\n") {
		var line map[string]any
		if err := json.Unmarshal([]byte(l), &line); err != nil {
			t.Fatalf("log line is not JSON: %q", l)
		}
		if line["msg"] == "handler panicked" {
			panicked = line
		} else {
			access = line
		}
	}
	if panicked == nil || panicked["correlation_id"] != "caller-9" || !strings.Contains(panicked["panic"].(string), "boom") {
		t.Fatalf("panic log line = %v", panicked)
	}
	if access == nil || access["status"] != float64(500) || access["level"] != "ERROR" {
		t.Fatalf("access log line = %v, want one ERROR line with status 500", access)
	}
}

// A panic after the response began cannot change the status, but must not
// crash the server or go unlogged.
func TestPanicAfterTheResponseBeganIsLoggedAndContained(t *testing.T) {
	var sink bytes.Buffer
	h := RequestLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("partial"))
		panic("late")
	}), slog.New(slog.NewJSONHandler(&sink, nil)))
	logged(t, h, "")
	if !strings.Contains(sink.String(), "handler panicked") {
		t.Fatalf("panic not logged: %s", sink.String())
	}
}

// The recorder must not hide the underlying writer's optional interfaces from
// http.ResponseController.
func TestRecorderUnwrapsToTheUnderlyingWriter(t *testing.T) {
	under := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: under}
	if rec.Unwrap() != http.ResponseWriter(under) {
		t.Fatal("Unwrap did not return the underlying writer")
	}
	if err := http.NewResponseController(rec).Flush(); err != nil {
		t.Fatalf("Flush through the recorder: %v", err)
	}
	if !under.Flushed {
		t.Fatal("flush did not reach the underlying writer")
	}
}
