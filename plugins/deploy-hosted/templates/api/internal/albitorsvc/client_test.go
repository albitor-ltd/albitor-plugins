package albitorsvc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(t *testing.T, srv *httptest.Server, sleeps *[]time.Duration) *Client {
	t.Helper()
	c, err := New(Config{
		BaseURL:     srv.URL,
		Region:      "eu-west-2",
		Credentials: Credentials{AccessKeyID: "AKID", SecretAccessKey: "s", SessionToken: "tok"},
		HTTP:        srv.Client(),
		Now:         func() time.Time { return time.Date(2026, 10, 29, 1, 2, 3, 0, time.UTC) },
		Sleep: func(_ context.Context, d time.Duration) error {
			if sleeps != nil {
				*sleeps = append(*sleeps, d)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func refuse(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `"}`))
}

func TestCallSendsASignedAWSShapedPost(t *testing.T) {
	var gotPath, gotAuth, gotKey, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotPath, gotAuth, gotKey, gotBody = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Idempotency-Key"), string(b)
		_, _ = w.Write([]byte(`{"TranslatedText":"bonjour"}`))
	}))
	defer srv.Close()

	var resp struct{ TranslatedText string }
	err := testClient(t, srv, nil).Call(context.Background(), "translate", "TranslateText",
		map[string]string{"Text": "hello", "SourceLanguageCode": "en", "TargetLanguageCode": "fr"}, &resp,
		WithIdempotencyKey("k-1"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.TranslatedText != "bonjour" {
		t.Errorf("response = %+v", resp)
	}
	if gotPath != "/v1/translate/TranslateText" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotAuth, "/eu-west-2/lambda/aws4_request") {
		t.Errorf("Authorization %q is not scoped for lambda in eu-west-2", gotAuth)
	}
	if gotKey != "k-1" {
		t.Errorf("Idempotency-Key = %q", gotKey)
	}
	var sent map[string]string
	if err := json.Unmarshal([]byte(gotBody), &sent); err != nil || sent["Text"] != "hello" {
		t.Errorf("body = %q", gotBody)
	}
}

func TestCallTypesTheGatewayRefusals(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		code    string
		is      []error
		isNot   []error
		unavail bool
	}{
		{"credits", 402, "credits_exhausted", []error{ErrCreditsExhausted}, []error{ErrCapReached, ErrForbidden}, true},
		{"cap", 402, "cap_reached", []error{ErrCapReached}, []error{ErrCreditsExhausted}, true},
		{"not declared", 403, "not_declared", []error{ErrForbidden}, []error{ErrCreditsExhausted}, true},
		{"switched off", 403, "service_disabled", []error{ErrForbidden}, nil, true},
		{"unavailable", 503, "unavailable", nil, []error{ErrForbidden, ErrRateLimited}, false},
		{"bad request", 400, "bad_request", nil, []error{ErrForbidden}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				refuse(w, tc.status, tc.code)
			}))
			defer srv.Close()
			err := testClient(t, srv, nil).Call(context.Background(), "translate", "TranslateText", nil, nil)
			var ge *Error
			if !errors.As(err, &ge) || ge.Status != tc.status || ge.Code != tc.code {
				t.Fatalf("err = %v", err)
			}
			for _, want := range tc.is {
				if !errors.Is(err, want) {
					t.Errorf("errors.Is(%v) = false", want)
				}
			}
			for _, not := range tc.isNot {
				if errors.Is(err, not) {
					t.Errorf("errors.Is(%v) = true", not)
				}
			}
			if Unavailable(err) != tc.unavail {
				t.Errorf("Unavailable = %v", !tc.unavail)
			}
			if calls.Load() != 1 {
				t.Errorf("a %d was retried: %d calls", tc.status, calls.Load())
			}
		})
	}
}

func TestCallRetries429WithJitterCappedAndHonouringRetryAfter(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.Header().Set("Retry-After", "2")
			refuse(w, 429, "rate_limited")
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	var sleeps []time.Duration
	if err := testClient(t, srv, &sleeps).Call(context.Background(), "translate", "TranslateText", nil, nil); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || len(sleeps) != 2 {
		t.Fatalf("calls = %d, sleeps = %v", calls.Load(), sleeps)
	}
	for _, d := range sleeps {
		if d < 2*time.Second {
			t.Errorf("slept %v, less than Retry-After", d)
		}
		if d > 5*time.Second {
			t.Errorf("slept %v, over the cap", d)
		}
	}
}

func TestCallGivesUpOn429AndCapsAHostileRetryAfter(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "86400")
		refuse(w, 429, "rate_limited")
	}))
	defer srv.Close()

	var sleeps []time.Duration
	err := testClient(t, srv, &sleeps).Call(context.Background(), "translate", "TranslateText", nil, nil)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 4 {
		t.Errorf("calls = %d, want the default 4 attempts", calls.Load())
	}
	for _, d := range sleeps {
		if d > 5*time.Second {
			t.Errorf("slept %v: Retry-After was not capped", d)
		}
	}
}

func TestCallStopsRetryingWhenTheContextEnds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { refuse(w, 429, "rate_limited") }))
	defer srv.Close()
	c := testClient(t, srv, nil)
	c.cfg.Sleep = func(ctx context.Context, _ time.Duration) error { return context.Canceled }
	err := c.Call(context.Background(), "translate", "TranslateText", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestCallRefusesAnUnsafePathBeforeSending(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer srv.Close()
	c := testClient(t, srv, nil)
	for _, bad := range [][2]string{{"translate", "../TranslateText"}, {"Translate", "TranslateText"}, {"translate/x", "TranslateText"}, {"translate", "translateText"}, {"", ""}} {
		if err := c.Call(context.Background(), bad[0], bad[1], nil, nil); err == nil {
			t.Errorf("%v was accepted", bad)
		}
	}
	if calls.Load() != 0 {
		t.Error("a refused name was sent")
	}
}

func TestErrorTextCarriesNoUpstreamMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(502)
		_, _ = w.Write([]byte(`{"error":"upstream_rejected","aws_code":"ValidationException","request_id":"r-1","message":"echoes the input"}`))
	}))
	defer srv.Close()
	err := testClient(t, srv, nil).Call(context.Background(), "translate", "TranslateText", nil, nil)
	var ge *Error
	if !errors.As(err, &ge) || ge.AWSCode != "ValidationException" || ge.RequestID != "r-1" {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "echoes") {
		t.Errorf("error text %q carries upstream text", err)
	}
}

func TestOpenReadsTheEnvironment(t *testing.T) {
	if _, err := Open(func(string) string { return "" }); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("unset URL: err = %v", err)
	}
	env := map[string]string{
		URLEnv:                  "https://abc.lambda-url.eu-west-2.on.aws/",
		"AWS_ACCESS_KEY_ID":     "AKID",
		"AWS_SECRET_ACCESS_KEY": "s",
	}
	c, err := Open(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if c.region != "eu-west-2" {
		t.Errorf("region = %q, want it taken from the function URL", c.region)
	}
	delete(env, "AWS_SECRET_ACCESS_KEY")
	if _, err := Open(func(k string) string { return env[k] }); err == nil {
		t.Error("a client was built with no credentials")
	}
}

func TestNewRefusesAPlainHTTPGatewayAndAnyRedirect(t *testing.T) {
	cfg := Config{BaseURL: "http://gateway.example.com/", Region: "eu-west-2", Credentials: Credentials{AccessKeyID: "a", SecretAccessKey: "s"}}
	if _, err := New(cfg); err == nil {
		t.Error("a plain-http, non-loopback gateway URL was accepted")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://elsewhere.invalid/", http.StatusFound)
	}))
	defer srv.Close()
	err := testClient(t, srv, nil).Call(context.Background(), "translate", "TranslateText", nil, nil)
	var ge *Error
	if !errors.As(err, &ge) || ge.Status != http.StatusFound {
		t.Errorf("a redirect was followed or hidden: %v", err)
	}
}
