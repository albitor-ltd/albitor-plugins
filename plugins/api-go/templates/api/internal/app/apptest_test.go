package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// This file is the shared test helper for the package. Features do not edit it: a feature's
// tests live in <feature>_test.go and call serve. Needing something it lacks is a reason to add a
// helper in your own test file, not to change this one.

// newMux returns a mux with every registered feature mounted.
func newMux(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	Routes(mux)
	return mux
}

// serve sends one request through the registered routes and returns the recorded response.
// body may be empty.
func serve(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
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
	newMux(t).ServeHTTP(rec, req)
	return rec
}
