// Package apptest is the shared test helper for feature packages. A feature's tests live beside
// the feature (internal/<feature>/<feature>_test.go) and mount only that feature's routes:
//
//	rec := apptest.Serve(t, Register, http.MethodGet, "/api/widgets", "")
//
// It imports nothing of the app, so every feature package can use it without an import cycle.
// Features do not edit this package; a helper only one feature needs goes in that feature's own
// _test.go file, and one several features need goes here only when it is about HTTP, never about
// any feature's data.
package apptest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Serve mounts one feature's register function on a fresh mux, sends one request (body may be
// empty) and returns the recorded response. A pattern the feature registers twice panics here,
// as it does at start.
func Serve(t *testing.T, register func(*http.ServeMux), method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	register(mux)
	return Do(t, mux, method, path, body)
}

// Do sends one request to h and returns the recorded response. body may be empty.
func Do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
