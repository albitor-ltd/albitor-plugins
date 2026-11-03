// Package supplier is a worked example of calling an external supplier's HTTP API. Rename the
// package, the type and the fields to the supplier's own, taking every field from its documented
// response.
package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// EnvBaseURL names the config entry holding the supplier's base URL. Declare it in albitor.yaml
// `config` (not secret) with the value the person gave. It has no default in code: a URL guessed
// from memory is a wrong host that reads like a network fault.
const EnvBaseURL = "SUPPLIER_BASE_URL"

// Lookup is the supplier's documented response, field for field. Add a field only when the
// supplier's documentation gives it. Both fields are required: a response without one is an
// error, not a zero value, so a gap shows on the first call rather than in the deployed walk.
type Lookup struct {
	Reference string `json:"reference"`
	Status    string `json:"status"`
}

// Client calls the supplier. Build it with New.
type Client struct {
	base string
	http *http.Client
}

// New reads the base URL from getenv (pass os.Getenv) and fails clearly when it is unset or not an
// absolute http(s) URL. Call it at start, or on first use, so a missing entry is named.
func New(getenv func(string) string) (*Client, error) {
	raw := strings.TrimRight(strings.TrimSpace(getenv(EnvBaseURL)), "/")
	if raw == "" {
		return nil, errors.New(EnvBaseURL + " is not set")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%s is not an absolute http(s) URL", EnvBaseURL)
	}
	return &Client{base: raw, http: &http.Client{Timeout: 10 * time.Second}}, nil
}

// Get fetches one record by reference. A 404 is ErrNotFound; any other non-200 is an error.
func (c *Client) Get(ctx context.Context, reference string) (Lookup, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/lookups/"+url.PathEscape(reference), nil)
	if err != nil {
		return Lookup{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Lookup{}, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return Lookup{}, ErrNotFound
	default:
		return Lookup{}, fmt.Errorf("supplier answered %d", resp.StatusCode)
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Lookup{}, fmt.Errorf("supplier response: %w", err)
	}
	for _, field := range []string{"reference", "status"} {
		if _, ok := raw[field]; !ok {
			return Lookup{}, fmt.Errorf("supplier response lacks %q", field)
		}
	}
	var l Lookup
	for field, dst := range map[string]*string{"reference": &l.Reference, "status": &l.Status} {
		if err := json.Unmarshal(raw[field], dst); err != nil {
			return Lookup{}, fmt.Errorf("supplier response field %q: %w", field, err)
		}
	}
	return l, nil
}

// ErrNotFound is the supplier's own "no such record", which is an answer, not a fault.
var ErrNotFound = errors.New("supplier has no such record")
