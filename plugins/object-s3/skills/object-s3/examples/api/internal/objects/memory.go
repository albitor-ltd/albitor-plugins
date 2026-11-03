package objects

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LocalPath is where a local run serves what Memory signs.
const LocalPath = "/api/local-objects/"

// Memory is the local run's store and the unit tests' fake. Its "signed" URLs
// point back at this API (Handler), which holds the PUT to the same rules S3
// does: exactly the declared length and content type, nothing else.
type Memory struct {
	mu      sync.Mutex
	objects map[string]stored
	pending map[string]PutSpec
	now     func() time.Time
}

type stored struct {
	data        []byte
	contentType string
	filename    string
}

// NewMemory returns an empty store.
func NewMemory() *Memory {
	return &Memory{objects: map[string]stored{}, pending: map[string]PutSpec{}, now: time.Now}
}

func localURL(key string) string { return LocalPath + escapeKey(key) }

func (m *Memory) PresignPut(_ context.Context, key string, spec PutSpec) (Signed, error) {
	if spec.Size <= 0 || spec.ContentType == "" {
		return Signed{}, ErrRefused
	}
	exp := clampExpiry(spec.Expires)
	m.mu.Lock()
	m.pending[key] = spec
	m.mu.Unlock()
	return Signed{
		Method: http.MethodPut,
		URL:    localURL(key),
		Headers: map[string]string{
			"Content-Type":        spec.ContentType,
			"Content-Disposition": contentDisposition(spec.Filename),
			"x-amz-meta-filename": url.PathEscape(spec.Filename),
		},
		ExpiresAt: m.now().Add(exp).UTC(),
	}, nil
}

func (m *Memory) PresignGet(_ context.Context, key string, expires time.Duration) (Signed, error) {
	return Signed{Method: http.MethodGet, URL: localURL(key), ExpiresAt: m.now().Add(clampExpiry(expires)).UTC()}, nil
}

func (m *Memory) Head(_ context.Context, key string) (Info, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objects[key]
	if !ok {
		return Info{}, ErrNotFound
	}
	return Info{Size: int64(len(o.data)), ContentType: o.contentType, Filename: o.filename}, nil
}

func (m *Memory) List(_ context.Context, prefix string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var keys []string
	for k := range m.objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (m *Memory) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.objects, k)
		delete(m.pending, k)
	}
	return nil
}

// Handler serves the local "signed" URLs. Register it for LocalPath only when
// Open returned a *Memory; a deployed function has no such route.
func (m *Memory) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.EscapedPath(), LocalPath)
		key, err := url.PathUnescape(key)
		if err != nil || key == "" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodPut:
			m.mu.Lock()
			spec, ok := m.pending[key]
			m.mu.Unlock()
			// S3 answers 403 to a PUT that does not match what was signed.
			if !ok || r.Header.Get("Content-Type") != spec.ContentType || r.ContentLength != spec.Size ||
				r.Header.Get("Content-Disposition") != contentDisposition(spec.Filename) {
				http.Error(w, "SignatureDoesNotMatch", http.StatusForbidden)
				return
			}
			data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, spec.Size))
			if err != nil || int64(len(data)) != spec.Size {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			name, _ := url.PathUnescape(r.Header.Get(metaFilename))
			m.mu.Lock()
			m.objects[key] = stored{data: data, contentType: spec.ContentType, filename: name}
			delete(m.pending, key)
			m.mu.Unlock()
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			m.mu.Lock()
			o, ok := m.objects[key]
			m.mu.Unlock()
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", o.contentType)
			w.Header().Set("Content-Disposition", contentDisposition(o.filename))
			w.Header().Set("Content-Length", strconv.Itoa(len(o.data)))
			_, _ = w.Write(o.data)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
