package outbound

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestForbiddenAddresses(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "127.1.2.3", "::1", "10.0.0.1", "172.16.0.1", "172.31.255.255", "192.168.1.1",
		"fd00:ec2::254", "fc00::1", "169.254.169.254", "169.254.0.1", "fe80::1", "0.0.0.0", "::",
		"224.0.0.1", "ff02::1", "ff01::1", "100.64.0.1", "198.18.0.1", "240.0.0.1", "255.255.255.255",
		"::ffff:127.0.0.1", "::ffff:10.0.0.1", "::ffff:169.254.169.254", "64:ff9b::a00:1",
	}
	for _, s := range blocked {
		if !Forbidden(netip.MustParseAddr(s)) {
			t.Errorf("%s should be forbidden", s)
		}
	}
	allowed := []string{"8.8.8.8", "93.184.216.34", "172.32.0.1", "11.0.0.1", "2606:4700:4700::1111", "::ffff:8.8.8.8"}
	for _, s := range allowed {
		if Forbidden(netip.MustParseAddr(s)) {
			t.Errorf("%s should be allowed", s)
		}
	}
	if !Forbidden(netip.Addr{}) {
		t.Error("an invalid address should be forbidden")
	}
}

func TestCheckRefusesBadURLs(t *testing.T) {
	c := New(Options{})
	bad := []string{
		"ftp://example.com/x", "file:///etc/passwd", "gopher://example.com", "//example.com/x",
		"http://", "http://user:pw@example.com/", "http://127.0.0.1/", "http://[::1]:8080/",
		"http://169.254.169.254/latest/meta-data/", "http://[fd00:ec2::254]/", "http://10.1.2.3/",
		"http://192.168.0.1:9000/", "http://0.0.0.0/", "http://[::ffff:10.0.0.1]/", "::not a url",
	}
	for _, raw := range bad {
		if _, err := c.Check(raw); !errors.Is(err, ErrBlocked) {
			t.Errorf("Check(%q) = %v, want ErrBlocked", raw, err)
		}
	}
	for _, raw := range []string{"https://example.com/x", "http://8.8.8.8/", "https://api.example.org:8443/a?b=c"} {
		if _, err := c.Check(raw); err != nil {
			t.Errorf("Check(%q) = %v, want nil", raw, err)
		}
	}
}

func TestAllowList(t *testing.T) {
	c := New(Options{AllowHosts: []string{"API.example.com.", "*.cdn.example.net"}})
	ok := []string{"https://api.example.com/x", "https://a.cdn.example.net/", "https://x.y.cdn.example.net/"}
	for _, raw := range ok {
		if _, err := c.Check(raw); err != nil {
			t.Errorf("Check(%q) = %v", raw, err)
		}
	}
	no := []string{
		"https://example.com/", "https://cdn.example.net/", "https://evilapi.example.com/",
		"https://api.example.com.evil.test/", "https://evil.test/?h=api.example.com",
	}
	for _, raw := range no {
		if _, err := c.Check(raw); !errors.Is(err, ErrBlocked) {
			t.Errorf("Check(%q) = %v, want ErrBlocked", raw, err)
		}
	}
}

func TestDialGuardRefusesLoopbackServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("the guarded client reached a loopback server")
	}))
	defer srv.Close()
	_, err := New(Options{}).Get(context.Background(), srv.URL)
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("err = %v, want ErrBlocked", err)
	}
}

// A name is judged on the address it resolves to, not on its spelling: localhost passes the
// URL check and is refused when dialled.
func TestDialGuardJudgesTheResolvedAddress(t *testing.T) {
	c := New(Options{})
	if _, err := c.Check("http://localhost:1/"); err != nil {
		t.Fatalf("a hostname passes the URL check: %v", err)
	}
	_, err := c.Get(context.Background(), "http://localhost:1/")
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("err = %v, want ErrBlocked at dial", err)
	}
}

func TestGetReturnsStatusHeadersAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != "" {
			t.Error("unexpected proxy header")
		}
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()
	resp, err := New(Options{allowLoopback: true}).Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != http.StatusTeapot || string(resp.Body) != "hello" || resp.Header.Get("X-Test") != "yes" {
		t.Fatalf("response = %+v", resp)
	}
}

func TestBodyLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 100)))
	}))
	defer srv.Close()
	c := New(Options{allowLoopback: true, MaxBody: 99})
	if _, err := c.Get(context.Background(), srv.URL); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
	c = New(Options{allowLoopback: true, MaxBody: 100})
	if _, err := c.Get(context.Background(), srv.URL); err != nil {
		t.Fatalf("exactly the limit: %v", err)
	}
}

// redirectTo serves a 302 to target and records whether anything else was reached.
func redirectTo(target string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusFound)
	}))
}

func TestRedirectsAreRecheckedForTheAddress(t *testing.T) {
	for _, target := range []string{
		"http://169.254.169.254/latest/meta-data/", "http://[fd00:ec2::254]/", "http://10.0.0.5/",
		"http://[::ffff:169.254.169.254]/",
	} {
		srv := redirectTo(target)
		_, err := New(Options{allowLoopback: true}).Get(context.Background(), srv.URL)
		srv.Close()
		if !errors.Is(err, ErrBlocked) {
			t.Errorf("redirect to %s: err = %v, want ErrBlocked", target, err)
		}
	}
}

func TestRedirectsAreRecheckedForSchemeAndAllowList(t *testing.T) {
	srv := redirectTo("ftp://example.com/x")
	defer srv.Close()
	if _, err := New(Options{allowLoopback: true}).Get(context.Background(), srv.URL); !errors.Is(err, ErrBlocked) {
		t.Errorf("redirect to ftp: err = %v, want ErrBlocked", err)
	}

	// The first host is allowed (the test server's 127.0.0.1); the redirect leaves the list.
	other := redirectTo("https://elsewhere.example.com/")
	defer other.Close()
	host := strings.TrimPrefix(other.URL, "http://")
	host = host[:strings.LastIndex(host, ":")]
	c := New(Options{allowLoopback: true, AllowHosts: []string{host}})
	if _, err := c.Get(context.Background(), other.URL); !errors.Is(err, ErrBlocked) {
		t.Errorf("redirect off the allow-list: err = %v, want ErrBlocked", err)
	}
}

func TestRedirectsFollowedWithinTheLimit(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/end", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("end"))
	}))
	defer final.Close()
	resp, err := New(Options{allowLoopback: true}).Get(context.Background(), final.URL+"/start")
	if err != nil || string(resp.Body) != "end" {
		t.Fatalf("resp = %+v, err = %v", resp, err)
	}
	if _, err := New(Options{allowLoopback: true, MaxRedirects: -1}).Get(context.Background(), final.URL+"/start"); !errors.Is(err, ErrBlocked) {
		t.Errorf("no redirects allowed: err = %v, want ErrBlocked", err)
	}
}

func TestRedirectLoopIsStopped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.Path, http.StatusFound)
	}))
	defer srv.Close()
	if _, err := New(Options{allowLoopback: true, MaxRedirects: 3}).Get(context.Background(), srv.URL); !errors.Is(err, ErrBlocked) {
		t.Fatalf("err = %v, want ErrBlocked", err)
	}
}

func TestTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { <-release }))
	defer srv.Close()
	defer close(release)
	_, err := New(Options{allowLoopback: true, Timeout: 50 * time.Millisecond}).Get(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("want a timeout")
	}
}

func TestNoProxyFromTheEnvironment(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("http_proxy", "http://127.0.0.1:1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("direct")) }))
	defer srv.Close()
	resp, err := New(Options{allowLoopback: true}).Get(context.Background(), srv.URL)
	if err != nil || string(resp.Body) != "direct" {
		t.Fatalf("resp = %+v, err = %v", resp, err)
	}
}
