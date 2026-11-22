// Package outbound is the one way this API fetches a URL it did not write itself: a URL that
// arrives in a request body or query, in a stored record or in configuration. A bare http.Client
// fetches whatever address the URL names, so a client that sends http://169.254.169.254/ (the
// cloud metadata service) or http://localhost:8080/ turns the API into a way into the network it
// runs in (server-side request forgery). Fetch through Client instead.
//
// What it enforces:
//
//   - the scheme is http or https, and the URL carries no user:password;
//   - with Options.AllowHosts set, the host is one of those names (or under a "*.example.com"
//     entry), checked again on every redirect;
//   - at dial time, the address actually connected to is refused when it is loopback, private
//     (RFC 1918, IPv6 unique local, which includes fd00:ec2::254), link-local (which includes
//     169.254.169.254), unspecified, multicast or another reserved range. The check is on the
//     resolved IP, so a name that resolves to a private address (or is rebound to one between a
//     check and the connection) is refused, as is a redirect to one. No proxy from the
//     environment is used, because a proxy would be dialled instead of the target.
//
// Standard library only.
package outbound

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// ErrBlocked is wrapped by every refusal: a scheme that is not http or https, a host outside the
// allow-list, a forbidden address, too many redirects. Answer the caller with a 400 (the URL
// they gave is not acceptable), never with the detail.
var ErrBlocked = errors.New("outbound: URL not allowed")

// ErrTooLarge is returned when a response body is longer than Options.MaxBody.
var ErrTooLarge = errors.New("outbound: response body too large")

// Options configures a Client. The zero value is valid.
type Options struct {
	// AllowHosts, when not empty, limits the fetch to these hosts: an exact name
	// ("api.example.com") or a suffix entry ("*.example.com", which matches subdomains and not
	// example.com itself). Matching ignores case and a trailing dot. Empty means any host that
	// resolves to a public address. Set it whenever the set of hosts is known; the address guard
	// is a backstop, not a substitute.
	AllowHosts []string
	// Timeout bounds one whole fetch, redirects included. Default 10 seconds.
	Timeout time.Duration
	// MaxRedirects is how many redirects are followed. Default 5; a negative value follows none.
	MaxRedirects int
	// MaxBody is the most response bytes Get returns. Default 5 MiB.
	MaxBody int64

	// allowLoopback lets the tests in this package serve from httptest on 127.0.0.1. It is
	// unexported so no caller can switch the guard off.
	allowLoopback bool
}

// Client fetches URLs under the rules in the package comment.
type Client struct {
	http *http.Client
	opts Options
}

// Response is what Get returns. Any status is returned, not only 2xx; check Status.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// New returns a Client.
func New(o Options) *Client {
	if o.Timeout <= 0 {
		o.Timeout = 10 * time.Second
	}
	if o.MaxRedirects == 0 {
		o.MaxRedirects = 5
	}
	if o.MaxBody <= 0 {
		o.MaxBody = 5 << 20
	}
	c := &Client{opts: o}
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		// Control runs once per address, after DNS and before the connection is made, so it
		// sees the IP that would really be connected to.
		Control: c.control,
	}
	c.http = &http.Client{
		Timeout: o.Timeout,
		Transport: &http.Transport{
			Proxy:               nil,
			DialContext:         dialer.DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
		},
		CheckRedirect: c.checkRedirect,
	}
	return c
}

// HTTPClient returns the guarded *http.Client for a call Get does not cover (a POST, a header).
// Build the request from a URL Check has accepted; the redirect and dial checks still apply.
func (c *Client) HTTPClient() *http.Client { return c.http }

// Check reports whether rawURL may be fetched, before any connection: http or https, a host, no
// credentials, a host the allow-list accepts, and an IP literal that is not forbidden. It returns
// the parsed URL. A name is judged again at dial time on the address it resolves to.
func (c *Client) Check(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: not a URL", ErrBlocked)
	}
	return u, c.checkURL(u)
}

// Get fetches rawURL with a GET and returns the status, headers and at most MaxBody bytes.
func (c *Client) Get(ctx context.Context, rawURL string) (*Response, error) {
	u, err := c.Check(rawURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBlocked, err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.opts.MaxBody+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > c.opts.MaxBody {
		return nil, ErrTooLarge
	}
	return &Response{Status: resp.StatusCode, Header: resp.Header, Body: body}, nil
}

func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	if c.opts.MaxRedirects < 0 || len(via) > c.opts.MaxRedirects {
		return fmt.Errorf("%w: too many redirects", ErrBlocked)
	}
	return c.checkURL(req.URL)
}

func (c *Client) checkURL(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: scheme must be http or https", ErrBlocked)
	}
	if u.User != nil {
		return fmt.Errorf("%w: credentials in the URL", ErrBlocked)
	}
	host := normaliseHost(u.Hostname())
	if host == "" {
		return fmt.Errorf("%w: no host", ErrBlocked)
	}
	if len(c.opts.AllowHosts) > 0 && !hostAllowed(host, c.opts.AllowHosts) {
		return fmt.Errorf("%w: host is not allowed", ErrBlocked)
	}
	if ip, err := netip.ParseAddr(host); err == nil && c.forbidden(ip) {
		return fmt.Errorf("%w: address is not allowed", ErrBlocked)
	}
	return nil
}

// control is the dial-time guard: address is the "ip:port" about to be connected to.
func (c *Client) control(network, address string, _ syscall.RawConn) error {
	if network != "tcp4" && network != "tcp6" {
		return fmt.Errorf("%w: network %q", ErrBlocked, network)
	}
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("%w: unreadable address", ErrBlocked)
	}
	if c.forbidden(ap.Addr()) {
		return fmt.Errorf("%w: address is not allowed", ErrBlocked)
	}
	return nil
}

func (c *Client) forbidden(ip netip.Addr) bool {
	if c.opts.allowLoopback && ip.Unmap().IsLoopback() {
		return false
	}
	return Forbidden(ip)
}

// reserved are ranges the netip predicates do not cover: this-network, carrier-grade NAT,
// IETF protocol and benchmarking blocks, the reserved class E (with the broadcast address) and
// the NAT64 prefixes, which can embed any IPv4 address.
var reserved = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
}

// Forbidden reports whether ip is an address an outbound fetch must never reach: loopback,
// private (RFC 1918 and IPv6 unique local), link-local (169.254.0.0/16, fe80::/10; both cloud
// metadata addresses 169.254.169.254 and fd00:ec2::254 fall here), unspecified, multicast or
// reserved. An IPv4-mapped IPv6 address is judged as the IPv4 address it carries.
func Forbidden(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() ||
		ip.IsUnspecified() {
		return true
	}
	for _, p := range reserved {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

func normaliseHost(h string) string {
	return strings.ToLower(strings.TrimSuffix(h, "."))
}

func hostAllowed(host string, allow []string) bool {
	for _, a := range allow {
		a = normaliseHost(a)
		if suffix, ok := strings.CutPrefix(a, "*."); ok {
			if suffix != "" && strings.HasSuffix(host, "."+suffix) {
				return true
			}
			continue
		}
		if a != "" && host == a {
			return true
		}
	}
	return false
}
