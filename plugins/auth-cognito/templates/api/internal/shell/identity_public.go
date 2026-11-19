package shell

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"webapp/internal/apikit"
)

// PUBLIC ROUTES (the browse posture). With AUTH_POSTURE=public the gate lets the API routes named
// in AUTH_PUBLIC_ROUTES through with no sign-in; everything else stays default-deny.
//
// AUTH_PUBLIC_ROUTES is services.auth.public_routes exactly as written in albitor.yaml, joined by
// commas. A web entry starts with "/" and is the web guard's business: the API gate ignores it. An
// API entry is "METHOD /api/path": "{param}" matches exactly one path segment and a trailing "/*"
// matches one or more further segments. The method is matched exactly (a public GET also answers
// HEAD). A malformed list closes the whole API, as every other variable here does.
//
// On a public route identity is optional: no credential means the handler runs with no identity
// (identity.FromContext says so; identity.Owner refuses in a deployed function, so a handler never
// acts for a shared owner), a valid credential is verified and its identity is used, and a
// credential that is present but invalid or expired answers 401, never a silent downgrade.
const (
	envPublicRoutes    = "AUTH_PUBLIC_ROUTES"
	envPublicBodyMax   = "AUTH_PUBLIC_BODY_MAX"
	postureDefault     = "private"
	posturePublic      = "public"
	defaultPublicBody  = 1 << 20
	maxPublicRoutes    = 50
	maxPublicRouteSize = 200
)

// publicRoute is one API entry of the list.
type publicRoute struct {
	method   string
	segments []string // literal, or "{}" for a parameter
	wildcard bool     // a trailing /*
}

// publicPolicy is the parsed posture. The zero value is private: nothing is public.
type publicPolicy struct {
	public  bool
	routes  []publicRoute
	bodyMax int64
}

var publicMethods = map[string]bool{
	http.MethodGet: true, http.MethodPost: true, http.MethodPut: true,
	http.MethodPatch: true, http.MethodDelete: true,
}

// parsePublicPolicy reads the three variables. Unset posture means private. Public needs a
// non-empty list; private takes none. Any malformed entry is an error.
func parsePublicPolicy(posture, rawRoutes, rawBodyMax string) (publicPolicy, error) {
	posture = strings.TrimSpace(posture)
	if posture == "" {
		posture = postureDefault
	}
	if posture != postureDefault && posture != posturePublic {
		return publicPolicy{}, fmt.Errorf("%s must be private or public", envPosture)
	}
	if posture == postureDefault {
		if strings.TrimSpace(rawRoutes) != "" {
			return publicPolicy{}, fmt.Errorf("%s is set but %s is private", envPublicRoutes, envPosture)
		}
		return publicPolicy{}, nil
	}
	p := publicPolicy{public: true, bodyMax: defaultPublicBody}
	if s := strings.TrimSpace(rawBodyMax); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n <= 0 {
			return publicPolicy{}, fmt.Errorf("%s must be a positive number of bytes", envPublicBodyMax)
		}
		p.bodyMax = n
	}
	if strings.TrimSpace(rawRoutes) == "" {
		return publicPolicy{}, fmt.Errorf("%s must list at least one route when %s is public", envPublicRoutes, envPosture)
	}
	entries := strings.Split(rawRoutes, ",")
	if len(entries) > maxPublicRoutes {
		return publicPolicy{}, fmt.Errorf("%s holds more than %d entries", envPublicRoutes, maxPublicRoutes)
	}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" || len(entry) > maxPublicRouteSize {
			return publicPolicy{}, fmt.Errorf("%s holds an empty or over-long entry", envPublicRoutes)
		}
		if strings.HasPrefix(entry, "/") {
			if strings.ContainsAny(entry, " \t\r\n") {
				return publicPolicy{}, fmt.Errorf("%s holds %q, a web path with a space in it", envPublicRoutes, entry)
			}
			continue // a web path: the web guard's, not the API's
		}
		route, err := parsePublicRoute(entry)
		if err != nil {
			return publicPolicy{}, err
		}
		p.routes = append(p.routes, route)
	}
	return p, nil
}

func parsePublicRoute(entry string) (publicRoute, error) {
	bad := func(why string) (publicRoute, error) {
		return publicRoute{}, fmt.Errorf("%s holds %q, which is not a web path or METHOD /api/path (%s)", envPublicRoutes, entry, why)
	}
	method, path, ok := strings.Cut(entry, " ")
	if !ok || !publicMethods[method] {
		return bad("the method must be GET, POST, PUT, PATCH or DELETE")
	}
	path = strings.TrimSpace(path)
	rest, ok := strings.CutPrefix(path, "/api/")
	if !ok || rest == "" {
		return bad("the path must start with /api/")
	}
	r := publicRoute{method: method}
	segs := strings.Split(rest, "/")
	for i, seg := range segs {
		switch {
		case seg == "*" && i == len(segs)-1 && i > 0:
			r.wildcard = true
		case seg == "*":
			return bad("a wildcard must be the last segment, after a literal one")
		case strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") && len(seg) > 2 && apikit.IsSafeToken(seg[1:len(seg)-1], 64):
			r.segments = append(r.segments, "{}")
		case seg != "." && seg != ".." && isLiteralSegment(seg):
			r.segments = append(r.segments, seg)
		default:
			return bad("a segment is empty or holds a character a path segment may not")
		}
	}
	if len(r.segments) == 0 {
		return bad("the path has no literal segment")
	}
	return r, nil
}

func isLiteralSegment(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '.', c == '~':
		default:
			return false
		}
	}
	return true
}

// isPublic reports whether the request matches a listed API route.
func (p publicPolicy) isPublic(r *http.Request) bool {
	if !p.public {
		return false
	}
	method := r.Method
	if method == http.MethodHead {
		method = http.MethodGet
	}
	rest, ok := strings.CutPrefix(r.URL.Path, "/api/")
	if !ok || rest == "" {
		return false
	}
	got := strings.Split(rest, "/")
	for _, route := range p.routes {
		if route.method == method && route.matches(got) {
			return true
		}
	}
	return false
}

func (route publicRoute) matches(got []string) bool {
	if route.wildcard {
		if len(got) <= len(route.segments) {
			return false
		}
	} else if len(got) != len(route.segments) {
		return false
	}
	for _, seg := range got {
		if seg == "" {
			return false
		}
	}
	for i, want := range route.segments {
		if want != "{}" && want != got[i] {
			return false
		}
	}
	return true
}

// capPublicBody refuses a public write whose body is over the cap with 413, and bounds a body of
// unknown length so a handler reading past the cap gets an error. It reports whether to go on.
func (p publicPolicy) capPublicBody(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	if r.ContentLength > p.bodyMax {
		apikit.WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return false
	}
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, p.bodyMax)
	}
	return true
}

// hasCredential reports whether the request offers a token at all: an Authorization header or a
// session cookie with a value. Whether it is any good is authenticate's question.
func hasCredential(r *http.Request) bool {
	if r.Header.Get("Authorization") != "" {
		return true
	}
	c, err := r.Cookie(sessionCookie)
	return err == nil && c.Value != ""
}
