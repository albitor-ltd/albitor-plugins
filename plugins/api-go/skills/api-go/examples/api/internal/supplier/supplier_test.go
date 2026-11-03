package supplier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// documentedResponse is the supplier's documented 200 body for GET /lookups/{reference}, copied
// from its documentation: exactly the documented fields and no others. Do not add a field to
// make a test or a screen pass. If the app needs a value the supplier does not return, that is
// a gap in the plan to report, and the fixture stays as documented.
const documentedResponse = `{"reference":"R-100","status":"active"}`

func envOf(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestNewFailsClearlyWhenTheBaseURLIsUnset(t *testing.T) {
	for _, env := range []map[string]string{{}, {EnvBaseURL: "  "}} {
		if _, err := New(envOf(env)); err == nil || !strings.Contains(err.Error(), EnvBaseURL+" is not set") {
			t.Errorf("env %v: err = %v, want %q named", env, err, EnvBaseURL+" is not set")
		}
	}
	if _, err := New(envOf(map[string]string{EnvBaseURL: "example.test/api"})); err == nil {
		t.Error("a base URL with no scheme was accepted")
	}
}

func TestTheFixtureHasOnlyTheDocumentedFields(t *testing.T) {
	dec := json.NewDecoder(bytes.NewReader([]byte(documentedResponse)))
	dec.DisallowUnknownFields()
	var l Lookup
	if err := dec.Decode(&l); err != nil {
		t.Fatalf("the fixture carries a field Lookup (the documented response) lacks: %v", err)
	}
}

func serve(t *testing.T, status int, body string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lookups/R-100" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c, err := New(envOf(map[string]string{EnvBaseURL: srv.URL + "/"}))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestGetReadsTheDocumentedResponse(t *testing.T) {
	got, err := serve(t, http.StatusOK, documentedResponse).Get(context.Background(), "R-100")
	if err != nil {
		t.Fatal(err)
	}
	if got != (Lookup{Reference: "R-100", Status: "active"}) {
		t.Errorf("got %+v", got)
	}
}

func TestAMissingDocumentedFieldIsAnErrorNotAZeroValue(t *testing.T) {
	_, err := serve(t, http.StatusOK, `{"reference":"R-100"}`).Get(context.Background(), "R-100")
	if err == nil || !strings.Contains(err.Error(), `"status"`) {
		t.Errorf("err = %v, want the missing field named", err)
	}
}

func TestNotFoundIsAnAnswer(t *testing.T) {
	_, err := serve(t, http.StatusNotFound, "").Get(context.Background(), "R-100")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
