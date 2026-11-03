package objects

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The AWS documentation's worked example of a presigned GET (Signature
// Version 4, query string): the signature must match byte for byte.
func TestPresignMatchesAWSDocumentationVector(t *testing.T) {
	s, err := NewS3(S3Config{
		Bucket: "examplebucket",
		Region: "us-east-1",
		Host:   "examplebucket.s3.amazonaws.com",
		Credentials: Credentials{
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
		Now: func() time.Time { return time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	got := s.url(s.presign(http.MethodGet, "test.txt", nil, nil, 86400*time.Second))
	want := "https://examplebucket.s3.amazonaws.com/test.txt?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
		"&X-Amz-Credential=AKIAIOSFODNN7EXAMPLE%2F20130524%2Fus-east-1%2Fs3%2Faws4_request" +
		"&X-Amz-Date=20130524T000000Z&X-Amz-Expires=86400&X-Amz-SignedHeaders=host" +
		"&X-Amz-Signature=aeeed9bbccd4d02ee5c0109b86d86835f995330da4c265957d157751f604d404"
	if got != want {
		t.Fatalf("presigned URL\n got %s\nwant %s", got, want)
	}
}

func TestPresignPutSignsLengthTypeAndExpiry(t *testing.T) {
	s, _ := NewS3(S3Config{
		Bucket: "b", Region: "eu-west-2", Prefix: "_objects/uploads/",
		Credentials: Credentials{AccessKeyID: "AK", SecretAccessKey: "SK", SessionToken: "TOK"},
		Now:         func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
	})
	out, err := s.PresignPut(context.Background(), "u1/r1/f1", PutSpec{ContentType: "image/png", Size: 1234, Filename: "my pic.png", Expires: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.URL, "/_objects/uploads/u1/r1/f1?") {
		t.Fatalf("the browser must get a path on the app origin, mapped 1:1 to the key: %s", out.URL)
	}
	for _, want := range []string{"content-length%3Bcontent-type%3Bhost%3Bx-amz-meta-filename", "X-Amz-Expires=900", "X-Amz-Security-Token=TOK"} {
		if !strings.Contains(out.URL, want) {
			t.Errorf("URL lacks %q: %s", want, out.URL)
		}
	}
	if out.Headers["x-amz-meta-filename"] != "my%20pic.png" || out.Headers["Content-Type"] != "image/png" {
		t.Errorf("headers = %v", out.Headers)
	}
	if _, ok := out.Headers["Content-Length"]; ok {
		t.Error("the browser sets Content-Length; it must not be returned as a header to set")
	}
}

// verify is what S3 does with a request: rebuild the canonical request from the
// request as received (method, path, query less the signature, the signed
// headers as sent) and compare signatures. It is written from the receiving
// side so it does not share the signer's code path.
func verify(t *testing.T, r *http.Request, host, region, secret string) bool {
	t.Helper()
	q := r.URL.Query()
	got := q.Get("X-Amz-Signature")
	q.Del("X-Amz-Signature")
	var pairs []string
	for k, vs := range q {
		for _, v := range vs {
			pairs = append(pairs, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	sort.Strings(pairs)
	canonicalQ := strings.NewReplacer("+", "%20", "%7E", "~").Replace(strings.Join(pairs, "&"))
	var hdr string
	for _, n := range strings.Split(q.Get("X-Amz-SignedHeaders"), ";") {
		v := r.Header.Get(n)
		switch n {
		case "host":
			v = host
		case "content-length":
			v = strconv.FormatInt(r.ContentLength, 10)
		}
		hdr += n + ":" + v + "\n"
	}
	path := strings.NewReplacer("+", "%20", "%2F", "/").Replace(url.QueryEscape(r.URL.Path))
	canon := strings.Join([]string{r.Method, path, canonicalQ, hdr, q.Get("X-Amz-SignedHeaders"), "UNSIGNED-PAYLOAD"}, "\n")
	date := q.Get("X-Amz-Date")
	scope := date[:8] + "/" + region + "/s3/aws4_request"
	sum := sha256.Sum256([]byte(canon))
	sts := "AWS4-HMAC-SHA256\n" + date + "\n" + scope + "\n" + hex.EncodeToString(sum[:])
	mac := func(k []byte, d string) []byte { h := hmac.New(sha256.New, k); h.Write([]byte(d)); return h.Sum(nil) }
	k := mac([]byte("AWS4"+secret), date[:8])
	k = mac(mac(mac(k, region), "s3"), "aws4_request")
	return hex.EncodeToString(mac(k, sts)) == got
}

// A presigned PUT with a security token, a signed Content-Length, Content-Type
// and metadata header, a key needing escapes, and a dotted bucket name: the
// request as the browser sends it verifies, and tampering with any signed
// part does not.
func TestPresignedPutRoundTrips(t *testing.T) {
	for _, bucket := range []string{"albitor-hosted-app--uploads", "my.dotted.bucket"} {
		s, _ := NewS3(S3Config{
			Bucket: bucket, Region: "eu-west-2", Prefix: "_objects/uploads/",
			Credentials: Credentials{AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "secret/KEY+1", SessionToken: "tok/en+="},
			Now:         func() time.Time { return time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC) },
		})
		host := bucket + ".s3.eu-west-2.amazonaws.com"
		out, err := s.PresignPut(context.Background(), "us-er/r 1/f~1", PutSpec{ContentType: "application/pdf", Size: 11, Filename: "Résumé (1).pdf"})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.URL, "X-Amz-Security-Token=tok%2Fen%2B%3D") {
			t.Errorf("token not escaped as SigV4 needs: %s", out.URL)
		}
		send := func(mutate func(*http.Request)) *http.Request {
			req := httptest.NewRequest(out.Method, out.URL, strings.NewReader("hello world"))
			for k, v := range out.Headers {
				req.Header.Set(k, v)
			}
			mutate(req)
			return req
		}
		if !verify(t, send(func(*http.Request) {}), host, "eu-west-2", "secret/KEY+1") {
			t.Errorf("%s: the browser's request does not verify", bucket)
		}
		for name, m := range map[string]func(*http.Request){
			"type":   func(r *http.Request) { r.Header.Set("Content-Type", "text/html") },
			"length": func(r *http.Request) { r.ContentLength = 12 },
			"name":   func(r *http.Request) { r.Header.Set("x-amz-meta-filename", "evil") },
			"key":    func(r *http.Request) { r.URL.Path = "/_objects/uploads/other/r1/f1" },
		} {
			if verify(t, send(m), host, "eu-west-2", "secret/KEY+1") {
				t.Errorf("%s: a changed %s still verifies", bucket, name)
			}
		}
	}
}

func TestSegmentMakesAnyIdSafe(t *testing.T) {
	if got := Segment("rec-17"); got != "rec-17" {
		t.Errorf("a safe id changed: %q", got)
	}
	for _, id := range []string{"REC#2026/17", "..", "a b", "../x", "ünï", strings.Repeat("a", 300), ""} {
		got := Segment(id)
		if _, err := Key("owner", got, "f"); err != nil {
			t.Errorf("Segment(%q) = %q is not a key segment: %v", id, got, err)
		}
		if got != Segment(id) {
			t.Errorf("Segment(%q) is not stable", id)
		}
	}
	if Segment("a/b") == Segment("a/c") {
		t.Error("distinct ids collided")
	}
}

func TestUnavailableFailsEveryCall(t *testing.T) {
	u := Unavailable(errors.New("no bucket"))
	if _, err := u.Head(context.Background(), "k"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Head: %v", err)
	}
	if err := u.Delete(context.Background(), "k"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Delete: %v", err)
	}
}

func TestPolicyRefusesBadSizeAndType(t *testing.T) {
	p := Policy{MaxSize: 10, Types: map[string]bool{"image/png": true}}
	for name, c := range map[string]struct {
		ct   string
		size int64
	}{"empty": {"image/png", 0}, "big": {"image/png", 11}, "type": {"text/html", 5}} {
		if err := p.Check(c.ct, c.size); !errors.Is(err, ErrRefused) {
			t.Errorf("%s: err = %v, want ErrRefused", name, err)
		}
	}
	if err := p.Check("IMAGE/PNG", 10); err != nil {
		t.Errorf("a match at the limit: %v", err)
	}
}

func TestKeyRefusesEscapes(t *testing.T) {
	for _, bad := range []string{"", "..", "a/b", "a..b", ".hidden", strings.Repeat("a", 129), "a b"} {
		if _, err := Key("owner", bad, "file"); err == nil {
			t.Errorf("record %q accepted", bad)
		}
	}
	if k, err := Key("us-er_1", "rec.1", NewID()); err != nil || !strings.HasPrefix(k, "us-er_1/rec.1/") {
		t.Fatalf("Key = %q, %v", k, err)
	}
}

// Memory is held to the rules S3 enforces on a signed PUT, so a handler that
// works in a local run works deployed.
func TestMemoryPutMustMatchWhatWasSigned(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	h := m.Handler()
	signed, _ := m.PresignPut(ctx, "o/r/f", PutSpec{ContentType: "image/png", Size: 3, Filename: "a b.png"})

	put := func(ct, body string) int {
		req := httptest.NewRequest(http.MethodPut, signed.URL, strings.NewReader(body))
		req.Header.Set("Content-Type", ct)
		req.Header.Set("x-amz-meta-filename", signed.Headers["x-amz-meta-filename"])
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := put("image/png", "toolong"); c != http.StatusForbidden {
		t.Errorf("wrong length: %d", c)
	}
	if c := put("text/html", "abc"); c != http.StatusForbidden {
		t.Errorf("wrong type: %d", c)
	}
	if _, err := m.Head(ctx, "o/r/f"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a refused PUT stored something: %v", err)
	}
	if c := put("image/png", "abc"); c != http.StatusOK {
		t.Fatalf("matching put: %d", c)
	}
	info, err := m.Head(ctx, "o/r/f")
	if err != nil || info.Size != 3 || info.Filename != "a b.png" {
		t.Fatalf("Head = %+v, %v", info, err)
	}
}

func TestDeleteUnderRemovesOnlyThePrefix(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	for _, k := range []string{"o/r1/a", "o/r1/b", "o/r2/a", "p/r1/a"} {
		s, _ := m.PresignPut(ctx, k, PutSpec{ContentType: "image/png", Size: 1})
		req := httptest.NewRequest(http.MethodPut, s.URL, strings.NewReader("x"))
		req.Header.Set("Content-Type", "image/png")
		m.Handler().ServeHTTP(httptest.NewRecorder(), req)
	}
	prefix, _ := Prefix("o", "r1")
	if err := DeleteUnder(ctx, m, prefix); err != nil {
		t.Fatal(err)
	}
	left, _ := m.List(ctx, "")
	if strings.Join(left, ",") != "o/r2/a,p/r1/a" {
		t.Fatalf("left = %v", left)
	}
	if err := DeleteUnder(ctx, m, prefix); err != nil {
		t.Fatalf("a second delete must be a no-op: %v", err)
	}
}

func TestOpenRefusesMemoryOnADeployedFunctionAndMissingBucket(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if _, err := Open(env(map[string]string{"ALBITOR_LOCAL": "1", "AWS_LAMBDA_FUNCTION_NAME": "f"}), Bucket); err == nil {
		t.Error("memory store on a deployed function")
	}
	deployed := map[string]string{"ALBITOR_BUCKET_UPLOADS": "b", "AWS_REGION": "eu-west-2", "AWS_ACCESS_KEY_ID": "a", "AWS_SECRET_ACCESS_KEY": "s"}
	if _, err := Open(env(deployed), Bucket); err == nil || !strings.Contains(err.Error(), "ALBITOR_OBJECTS_PATH_UPLOADS") {
		t.Errorf("missing origin path: %v", err)
	}
	deployed["ALBITOR_OBJECTS_PATH_UPLOADS"] = "/_objects/uploads/"
	if _, err := Open(env(deployed), Bucket); err != nil {
		t.Errorf("deployed: %v", err)
	}
	if _, err := Open(env(map[string]string{}), Bucket); err == nil || !strings.Contains(err.Error(), "ALBITOR_BUCKET_UPLOADS") {
		t.Errorf("missing bucket: %v", err)
	}
	if s, err := Open(env(map[string]string{"ALBITOR_LOCAL": "1"}), Bucket); err != nil {
		t.Errorf("local: %v", err)
	} else if _, ok := s.(*Memory); !ok {
		t.Errorf("local store is %T", s)
	}
}
