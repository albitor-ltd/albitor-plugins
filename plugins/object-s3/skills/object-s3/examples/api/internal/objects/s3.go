package objects

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Credentials are the function's execution-role credentials, as the Lambda
// runtime puts them in the environment.
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// S3Config configures an S3 store.
type S3Config struct {
	Bucket      string
	Region      string
	Credentials Credentials
	// Prefix is the object-key prefix the app origin maps to this bucket:
	// "_objects/<name>/", from ALBITOR_OBJECTS_PATH_<NAME>. Every key this
	// store touches lives under it, and the path the browser is given is the
	// key with a leading slash, which is what lets the app's own CloudFront
	// forward /_objects/<name>/* to the bucket unchanged.
	Prefix string
	// Host overrides <bucket>.s3.<region>.amazonaws.com. Tests only.
	Host   string
	Client *http.Client
	Now    func() time.Time
}

// S3 signs and calls S3 over the standard library, so the API module needs no
// dependency. Every operation is a query-signed URL. The browser gets the
// PUT and GET ones as a path and query on its own origin; this process calls
// the HEAD, LIST and DELETE ones itself, on the bucket's regional host.
//
// Why the signer is written out here and not an SDK: api-go's module has no
// dependencies, and a plugin may not add one. It does single-object requests
// only. Do not grow it: multipart upload, SSE-KMS, checksums and the like are
// the signal to move the app to a versioned module through a plugin major,
// not to extend this file.
type S3 struct {
	cfg S3Config
}

// MaxExpiry is the longest a signed URL may live. A URL signed with the
// function's temporary credentials stops working when they do, whatever the
// expiry says.
const MaxExpiry = 15 * time.Minute

const metaFilename = "x-amz-meta-filename"

// NewS3 builds the store, refusing a configuration that cannot sign.
func NewS3(cfg S3Config) (*S3, error) {
	switch {
	case cfg.Bucket == "":
		return nil, errors.New("objects: no bucket")
	case cfg.Region == "":
		return nil, errors.New("objects: no signing Region: set AWS_REGION")
	case cfg.Credentials.AccessKeyID == "" || cfg.Credentials.SecretAccessKey == "":
		return nil, errors.New("objects: no AWS credentials in the environment")
	}
	if cfg.Host == "" {
		cfg.Host = cfg.Bucket + ".s3." + cfg.Region + ".amazonaws.com"
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Prefix != "" && (!strings.HasPrefix(cfg.Prefix, "_objects/") || !strings.HasSuffix(cfg.Prefix, "/")) {
		return nil, fmt.Errorf("objects: prefix %q must be _objects/<name>/", cfg.Prefix)
	}
	return &S3{cfg: cfg}, nil
}

// url is the absolute URL for a path-and-query this store signed.
func (s *S3) url(rel string) string { return "https://" + s.cfg.Host + rel }

func clampExpiry(d time.Duration) time.Duration {
	if d <= 0 || d > MaxExpiry {
		return MaxExpiry
	}
	return d
}

// PresignPut signs a PUT with Content-Type, Content-Length, Content-Disposition
// (an attachment named for the file) and the file name as signed headers: S3 answers 403 to a body of another length or type.
func (s *S3) PresignPut(_ context.Context, key string, spec PutSpec) (Signed, error) {
	if spec.Size <= 0 || spec.ContentType == "" {
		return Signed{}, fmt.Errorf("%w: size and content type are required", ErrRefused)
	}
	exp := clampExpiry(spec.Expires)
	headers := map[string]string{
		"content-length": strconv.FormatInt(spec.Size, 10),
		"content-type":   spec.ContentType,
		metaFilename:     url.PathEscape(spec.Filename),
		// Stored with the object, so a GET needs no response-content-* override
		// (the app's CloudFront does not forward them): the download is an
		// attachment because the object says so.
		"content-disposition": contentDisposition(spec.Filename),
	}
	return Signed{
		Method: http.MethodPut,
		// A path on the app's own origin, never an S3 host: no CORS, no CSP change.
		URL: s.presign(http.MethodPut, s.cfg.Prefix+key, nil, headers, exp),
		// The browser sets Content-Length from the body.
		Headers: map[string]string{
			"Content-Type":        spec.ContentType,
			"Content-Disposition": headers["content-disposition"],
			"x-amz-meta-filename": headers[metaFilename],
		},
		ExpiresAt: s.cfg.Now().Add(exp).UTC(),
	}, nil
}

// PresignGet signs a plain GET. It carries no response-content-* parameters:
// the app's CloudFront may not forward them, which would fail the signature.
// Content-Type and Content-Disposition (attachment) were stored at upload, so an
// object is never rendered in the app's origin, whatever the bytes are.
func (s *S3) PresignGet(_ context.Context, key string, expires time.Duration) (Signed, error) {
	exp := clampExpiry(expires)
	return Signed{
		Method:    http.MethodGet,
		URL:       s.presign(http.MethodGet, s.cfg.Prefix+key, nil, nil, exp),
		ExpiresAt: s.cfg.Now().Add(exp).UTC(),
	}, nil
}

func contentDisposition(filename string) string {
	if filename == "" {
		return "attachment"
	}
	return "attachment; filename*=UTF-8''" + awsEscape(filename)
}

// Head reports an object, or ErrNotFound.
func (s *S3) Head(ctx context.Context, key string) (Info, error) {
	res, err := s.call(ctx, http.MethodHead, s.cfg.Prefix+key, nil)
	if err != nil {
		return Info{}, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		name, _ := url.PathUnescape(res.Header.Get(metaFilename))
		return Info{Size: res.ContentLength, ContentType: res.Header.Get("Content-Type"), Filename: name}, nil
	case http.StatusNotFound:
		return Info{}, ErrNotFound
	}
	return Info{}, fmt.Errorf("objects: head %s: status %d", key, res.StatusCode)
}

// Delete removes the keys one by one: DELETE of a missing key is a success,
// so a retry after a partial failure is safe. The bucket is versioned, so
// this leaves a delete marker and keeps the old version; see the skill.
func (s *S3) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		res, err := s.call(ctx, http.MethodDelete, s.cfg.Prefix+key, nil)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
			return fmt.Errorf("objects: delete %s: status %d", key, res.StatusCode)
		}
	}
	return nil
}

type listResult struct {
	Keys      []string `xml:"Contents>Key"`
	Truncated bool     `xml:"IsTruncated"`
	Next      string   `xml:"NextContinuationToken"`
}

// List returns every key under prefix, following continuation tokens.
func (s *S3) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	token := ""
	for {
		q := url.Values{"list-type": {"2"}, "prefix": {s.cfg.Prefix + prefix}}
		if token != "" {
			q.Set("continuation-token", token)
		}
		res, err := s.call(ctx, http.MethodGet, "", q)
		if err != nil {
			return nil, err
		}
		var out listResult
		err = func() error {
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("objects: list %s: status %d", prefix, res.StatusCode)
			}
			return xml.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(&out)
		}()
		if err != nil {
			return nil, err
		}
		for _, k := range out.Keys {
			keys = append(keys, strings.TrimPrefix(k, s.cfg.Prefix))
		}
		if !out.Truncated || out.Next == "" {
			return keys, nil
		}
		token = out.Next
	}
}

func (s *S3) call(ctx context.Context, method, key string, q url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.url(s.presign(method, key, q, nil, time.Minute)), nil)
	if err != nil {
		return nil, err
	}
	return s.cfg.Client.Do(req)
}

// presign returns the path and query ("/<key>?X-Amz-...") of a query-signed
// (SigV4) request for one method on one full object key. headers
// are lower-case names to values that the caller must send exactly as signed;
// host is always signed. The payload is UNSIGNED-PAYLOAD: a presigned URL
// cannot know the bytes.
func (s *S3) presign(method, key string, extra url.Values, headers map[string]string, expires time.Duration) string {
	c := s.cfg.Credentials
	now := s.cfg.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	day := now.Format("20060102")
	scope := day + "/" + s.cfg.Region + "/s3/aws4_request"

	all := map[string]string{"host": s.cfg.Host}
	for k, v := range headers {
		all[k] = v
	}
	names := make([]string, 0, len(all))
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)
	signed := strings.Join(names, ";")

	q := url.Values{}
	for k, v := range extra {
		q[k] = v
	}
	q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	q.Set("X-Amz-Credential", c.AccessKeyID+"/"+scope)
	q.Set("X-Amz-Date", amzDate)
	q.Set("X-Amz-Expires", strconv.FormatInt(int64(expires/time.Second), 10))
	q.Set("X-Amz-SignedHeaders", signed)
	if c.SessionToken != "" {
		q.Set("X-Amz-Security-Token", c.SessionToken)
	}
	query := canonicalQuery(q)

	var ch strings.Builder
	for _, n := range names {
		ch.WriteString(n + ":" + strings.Join(strings.Fields(all[n]), " ") + "\n")
	}
	path := "/" + escapeKey(key)
	canonical := strings.Join([]string{method, path, query, ch.String(), signed, "UNSIGNED-PAYLOAD"}, "\n")
	toSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + hexSHA256(canonical)

	k := hmacSHA256([]byte("AWS4"+c.SecretAccessKey), day)
	k = hmacSHA256(k, s.cfg.Region)
	k = hmacSHA256(k, "s3")
	k = hmacSHA256(k, "aws4_request")
	sig := hex.EncodeToString(hmacSHA256(k, toSign))

	return path + "?" + query + "&X-Amz-Signature=" + sig
}

func canonicalQuery(q url.Values) string {
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		for _, v := range q[k] {
			parts = append(parts, awsEscape(k)+"="+awsEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

// awsEscape percent-encodes everything but the unreserved characters, as SigV4
// requires (a space is %20, never +).
func awsEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_', c == '.', c == '~':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func escapeKey(key string) string {
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = awsEscape(p)
	}
	return strings.Join(parts, "/")
}

func hexSHA256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}
