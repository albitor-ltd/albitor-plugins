// Package objects is the app's object storage: private files that browsers
// upload and download directly, so the bytes never pass through the API
// function or the data store.
//
// The URLs are paths on the app's own origin (/_objects/<name>/...), which the
// app's CloudFront forwards to the bucket, so the browser makes same-origin
// requests: no CORS rule and no CSP change.
//
// The API only ever signs. It authorises a person for a key, then hands the
// browser a short-lived URL that allows one method on that one key. The store
// keeps a record's metadata; this package keeps its bytes.
//
// Deployed, a Store is one S3 bucket declared in albitor.yaml as
// services.objects. In a local run (ALBITOR_LOCAL=1) it is process memory and
// the API serves the "signed" URLs itself.
package objects

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Bucket is the logical name of the bucket declared in albitor.yaml
// (services.objects). The platform names the AWS bucket and hands it to the
// function as ALBITOR_BUCKET_UPLOADS.
const Bucket = "uploads"

var (
	// ErrNotFound is an object that is not there.
	ErrNotFound = errors.New("objects: not found")
	// ErrUnavailable is storage that is not configured or not reachable.
	ErrUnavailable = errors.New("objects: storage is unavailable")
	// ErrRefused is an upload the policy refuses: too big, empty or a type
	// that is not allowed.
	ErrRefused = errors.New("objects: refused")
)

// PutSpec is what an upload may be. Both fields are signed into the URL, so
// S3 itself refuses a body of another length or type.
type PutSpec struct {
	ContentType string
	Size        int64
	// Filename is the person's file name, kept as object metadata so a
	// download can offer it back. Any text; it is escaped before it is signed.
	Filename string
	Expires  time.Duration
}

// Signed is a URL the browser calls directly. Headers must be sent as given;
// the browser adds Content-Length itself from the body.
type Signed struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// Info is what is known about a stored object.
type Info struct {
	Size        int64
	ContentType string
	Filename    string
}

// Store is private object storage addressed by key.
type Store interface {
	// PresignPut signs a PUT of exactly spec.Size bytes of spec.ContentType.
	PresignPut(ctx context.Context, key string, spec PutSpec) (Signed, error)
	// PresignGet signs a GET. The object's own Content-Type and attachment
	// Content-Disposition, set at upload, shape the response.
	PresignGet(ctx context.Context, key string, expires time.Duration) (Signed, error)
	// Head reports an object, or ErrNotFound.
	Head(ctx context.Context, key string) (Info, error)
	// List returns every key under prefix.
	List(ctx context.Context, prefix string) ([]string, error)
	// Delete removes the keys. A key that is not there is not an error.
	Delete(ctx context.Context, keys ...string) error
}

// Policy is what the app accepts, decided before anything is signed.
type Policy struct {
	// MaxSize is the largest upload in bytes. It is the app's limit, not
	// S3's (5 GiB for one PUT); the API function's 6 MiB body limit does not
	// apply because the bytes never reach it.
	MaxSize int64
	// Types are the accepted content types, exact match, lower case. The
	// browser's claim is signed into the URL, so S3 refuses any other, but it
	// is still a claim: it is not a scan of the bytes.
	Types map[string]bool
}

// Check refuses a size or type the policy does not allow.
func (p Policy) Check(contentType string, size int64) error {
	switch {
	case size <= 0:
		return fmt.Errorf("%w: the file is empty", ErrRefused)
	case size > p.MaxSize:
		return fmt.Errorf("%w: the file is larger than %d bytes", ErrRefused, p.MaxSize)
	case !p.Types[strings.ToLower(contentType)]:
		return fmt.Errorf("%w: %q files are not accepted", ErrRefused, contentType)
	}
	return nil
}

var segmentRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// Keys here are logical: <owner>/<record>/<file>. The S3 store adds the
// "_objects/<name>/" prefix the origin path maps to, so no caller sees it.
//
// Key is <owner>/<record>/<file>. The owner comes from the verified identity
// and the file from NewID; the record comes from the URL, so each segment is
// checked: no slash, no dot segment, nothing that could escape the owner's
// prefix.
func Key(owner, record, file string) (string, error) {
	if err := checkSegments(owner, record, file); err != nil {
		return "", err
	}
	return owner + "/" + record + "/" + file, nil
}

// Prefix is <owner>/<record>/, every key a record owns. Pass an empty record
// for everything the owner has.
func Prefix(owner, record string) (string, error) {
	if record == "" {
		if err := checkSegments(owner); err != nil {
			return "", err
		}
		return owner + "/", nil
	}
	if err := checkSegments(owner, record); err != nil {
		return "", err
	}
	return owner + "/" + record + "/", nil
}

func checkSegments(segments ...string) error {
	for _, s := range segments {
		if !segmentRE.MatchString(s) || strings.Contains(s, "..") {
			return fmt.Errorf("objects: %q is not a key segment", s)
		}
	}
	return nil
}

// NewID is a file id: random, so it cannot be guessed and says nothing about
// the file. The person's file name is metadata, never part of the key.
func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // the system's random source failing is not recoverable
	}
	return hex.EncodeToString(b)
}

// Segment makes any record id a safe key segment. An id that already is one
// is returned as it is; anything else (a composite sort key such as
// "REC#2026/17", an email, a long or unicode id) becomes "h" and 32 hex
// characters of its SHA-256, so it is stable and cannot carry a slash or dot segment. Use the same function wherever the
// app turns a record id into a key, and keep the record's own id in the record.
func Segment(id string) string {
	if segmentRE.MatchString(id) && !strings.Contains(id, "..") {
		return id
	}
	sum := sha256.Sum256([]byte(id))
	return "h" + hex.EncodeToString(sum[:16])
}

// Unavailable is a Store whose every call fails with ErrUnavailable, for an
// app whose storage is not configured: the file routes answer 503 and the rest
// of the app stays up.
func Unavailable(cause error) Store { return unavailable{cause} }

type unavailable struct{ cause error }

func (u unavailable) err() error { return fmt.Errorf("%w: %v", ErrUnavailable, u.cause) }
func (u unavailable) PresignPut(context.Context, string, PutSpec) (Signed, error) {
	return Signed{}, u.err()
}
func (u unavailable) PresignGet(context.Context, string, time.Duration) (Signed, error) {
	return Signed{}, u.err()
}
func (u unavailable) Head(context.Context, string) (Info, error)     { return Info{}, u.err() }
func (u unavailable) List(context.Context, string) ([]string, error) { return nil, u.err() }
func (u unavailable) Delete(context.Context, ...string) error        { return u.err() }

// DeleteUnder removes every object under prefix: the files of a record that
// is being deleted. Call it before deleting the record, so a failure leaves
// the record to retry from rather than files nothing points at.
func DeleteUnder(ctx context.Context, s Store, prefix string) error {
	keys, err := s.List(ctx, prefix)
	if err != nil {
		return err
	}
	return s.Delete(ctx, keys...)
}
