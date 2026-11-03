package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"webapp/internal/identity"
	"webapp/internal/objects"
)

// owned is a stand-in for the data store lookup: record id to owner.
type owned map[string]string

func (o owned) owns(_ *http.Request, owner, record string) (bool, error) {
	return o[record] == owner, nil
}

func newFiles(records owned) (*Files, *objects.Memory, http.Handler) {
	mem := objects.NewMemory()
	f := &Files{
		Store:      mem,
		Policy:     objects.Policy{MaxSize: 1 << 20, Types: map[string]bool{"image/png": true, "application/pdf": true}},
		OwnsRecord: records.owns,
	}
	mux := http.NewServeMux()
	f.Routes(mux)
	mux.Handle(objects.LocalPath, mem.Handler())
	return f, mem, mux
}

func as(sub string, req *http.Request) *http.Request {
	return req.WithContext(identity.With(req.Context(), identity.Identity{Sub: sub}))
}

func do(h http.Handler, sub, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, as(sub, req))
	return rec
}

func startUpload(t *testing.T, h http.Handler, sub, record string, size int, ct string) (file string, signed objects.Signed) {
	t.Helper()
	body, _ := json.Marshal(startRequest{Name: "../../etc/passwd\x00.png", ContentType: ct, Size: int64(size)})
	rec := do(h, sub, http.MethodPost, "/api/records/"+record+"/files", string(body))
	if rec.Code != http.StatusCreated {
		t.Fatalf("start = %d %s", rec.Code, rec.Body)
	}
	var out startResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Name != "passwd.png" {
		t.Errorf("name = %q, want the cleaned base name", out.Name)
	}
	return out.FileID, out.Upload
}

func putBytes(h http.Handler, s objects.Signed, data string) int {
	req := httptest.NewRequest(s.Method, s.URL, bytes.NewReader([]byte(data)))
	for k, v := range s.Headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestUploadCompleteDownloadDelete(t *testing.T) {
	_, mem, h := newFiles(owned{"r1": "alice"})
	file, up := startUpload(t, h, "alice", "r1", 5, "image/png")

	if !strings.HasPrefix(up.URL, objects.LocalPath+"alice/r1/") {
		t.Fatalf("key is not under the owner and record: %s", up.URL)
	}
	if c := do(h, "alice", http.MethodPost, "/api/records/r1/files/"+file+"/complete", ""); c.Code != http.StatusConflict {
		t.Fatalf("complete before upload = %d", c.Code)
	}
	if c := putBytes(h, up, "12345"); c != http.StatusOK {
		t.Fatalf("put = %d", c)
	}
	rec := do(h, "alice", http.MethodPost, "/api/records/r1/files/"+file+"/complete", "")
	var d FileDetails
	_ = json.Unmarshal(rec.Body.Bytes(), &d)
	if rec.Code != http.StatusOK || d.Size != 5 || d.Name != "passwd.png" || d.ContentType != "image/png" {
		t.Fatalf("complete = %d %+v", rec.Code, d)
	}
	rec = do(h, "alice", http.MethodGet, "/api/records/r1/files/"+file+"?redirect=1", "")
	if rec.Code != http.StatusFound || !strings.HasPrefix(rec.Header().Get("Location"), objects.LocalPath+"alice/r1/") {
		t.Fatalf("download redirect = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if rec := do(h, "alice", http.MethodDelete, "/api/records/r1/files/"+file, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d", rec.Code)
	}
	if keys, _ := mem.List(context.Background(), ""); len(keys) != 0 {
		t.Fatalf("object left behind: %v", keys)
	}
}

func TestAnotherPersonCannotSignForARecordTheyDoNotOwn(t *testing.T) {
	_, mem, h := newFiles(owned{"r1": "alice"})
	body := `{"name":"a.png","content_type":"image/png","size":5}`
	if rec := do(h, "mallory", http.MethodPost, "/api/records/r1/files", body); rec.Code != http.StatusNotFound {
		t.Fatalf("upload to someone else's record = %d", rec.Code)
	}
	file, up := startUpload(t, h, "alice", "r1", 5, "image/png")
	putBytes(h, up, "12345")
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/records/r1/files/" + file},
		{http.MethodPost, "/api/records/r1/files/" + file + "/complete"},
		{http.MethodDelete, "/api/records/r1/files/" + file},
	} {
		if rec := do(h, "mallory", c.method, c.path, ""); rec.Code != http.StatusNotFound {
			t.Errorf("%s %s as another person = %d", c.method, c.path, rec.Code)
		}
	}
	// A file id guessed under a record id of their own still resolves inside
	// their own prefix, so it cannot reach alice's object.
	_, _, h2 := newFiles(owned{"r1": "mallory"})
	if rec := do(h2, "mallory", http.MethodGet, "/api/records/r1/files/"+file, ""); rec.Code != http.StatusNotFound {
		t.Errorf("same ids under another owner = %d", rec.Code)
	}
	if keys, _ := mem.List(context.Background(), "alice/"); len(keys) != 1 {
		t.Fatalf("alice's file was disturbed: %v", keys)
	}
}

func TestPolicyIsEnforcedBeforeAnythingIsSigned(t *testing.T) {
	_, _, h := newFiles(owned{"r1": "alice"})
	for name, body := range map[string]string{
		"too big":    `{"name":"a.png","content_type":"image/png","size":99999999}`,
		"wrong type": `{"name":"a.html","content_type":"text/html","size":5}`,
		"empty":      `{"name":"a.png","content_type":"image/png","size":0}`,
		"not json":   `nope`,
	} {
		rec := do(h, "alice", http.MethodPost, "/api/records/r1/files", body)
		if rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
	}
}

func TestRecordIdIsMadeASafeSegment(t *testing.T) {
	_, mem, h := newFiles(owned{"..": "alice", "REC#1/2": "alice"})
	for _, rid := range []string{"REC%231%2F2"} {
		rec := do(h, "alice", http.MethodPost, "/api/records/"+rid+"/files", `{"name":"a.png","content_type":"image/png","size":5}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("record %q = %d %s", rid, rec.Code, rec.Body)
		}
		var out startResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		if strings.Contains(out.Upload.URL, "REC") || strings.Contains(out.Upload.URL, "..") || !strings.Contains(out.Upload.URL, "alice/h") {
			t.Errorf("record id leaked into the key: %s", out.Upload.URL)
		}
		putBytes(h, out.Upload, "12345")
	}
	if keys, _ := mem.List(context.Background(), "alice/"); len(keys) != 1 {
		t.Fatalf("keys = %v", keys)
	}
}

func TestFailsClosedWithoutAnOwnershipCheck(t *testing.T) {
	f, _, _ := newFiles(owned{})
	f.OwnsRecord = nil
	mux := http.NewServeMux()
	f.Routes(mux)
	if rec := do(mux, "alice", http.MethodPost, "/api/records/r1/files", `{"name":"a.png","content_type":"image/png","size":5}`); rec.Code != http.StatusInternalServerError {
		t.Fatalf("no OwnsRecord = %d", rec.Code)
	}
}

func TestUnconfiguredStorageAnswers503(t *testing.T) {
	f, _, _ := newFiles(owned{"r1": "alice"})
	f.Store = objects.Unavailable(errors.New("ALBITOR_BUCKET_UPLOADS is not set"))
	mux := http.NewServeMux()
	f.Routes(mux)
	if rec := do(mux, "alice", http.MethodPost, "/api/records/r1/files", `{"name":"a.png","content_type":"image/png","size":5}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured = %d", rec.Code)
	}
}

func TestRedirectSendsNoReferrer(t *testing.T) {
	_, _, h := newFiles(owned{"r1": "alice"})
	file, up := startUpload(t, h, "alice", "r1", 1, "image/png")
	putBytes(h, up, "x")
	rec := do(h, "alice", http.MethodGet, "/api/records/r1/files/"+file+"?redirect=1", "")
	if rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("Referrer-Policy = %q", rec.Header().Get("Referrer-Policy"))
	}
}

func TestDeletingARecordDeletesItsFilesAndNoOthers(t *testing.T) {
	_, mem, h := newFiles(owned{"r1": "alice", "r2": "alice"})
	for _, rid := range []string{"r1", "r1", "r2"} {
		_, up := startUpload(t, h, "alice", rid, 1, "image/png")
		putBytes(h, up, "x")
	}
	req := as("alice", httptest.NewRequest(http.MethodDelete, "/api/records/r1", nil))
	if err := DeleteRecordFiles(req, mem, "r1"); err != nil {
		t.Fatal(err)
	}
	left, _ := mem.List(context.Background(), "")
	if len(left) != 1 || !strings.HasPrefix(left[0], "alice/r2/") {
		t.Fatalf("left = %v", left)
	}
}

func TestCleanName(t *testing.T) {
	for in, want := range map[string]string{
		"report.pdf": "report.pdf", `C:\temp\a.pdf`: "a.pdf", "": "file", "..": "file", "a\nb.png": "ab.png",
		strings.Repeat("x", 300): strings.Repeat("x", 200),
	} {
		if got := cleanName(in); got != want {
			t.Errorf("cleanName(%q) = %q, want %q", in, got, want)
		}
	}
}
