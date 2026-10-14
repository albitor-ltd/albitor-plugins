package store

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// call is one request the fake DynamoDB saw.
type call struct {
	Target string
	Auth   string
	Body   map[string]any
}

// fakeDynamo answers each request with the next scripted reply and records what it was sent.
type fakeDynamo struct {
	t       *testing.T
	replies []func(w http.ResponseWriter)
	calls   []call
	srv     *httptest.Server
}

func newFakeDynamo(t *testing.T, replies ...func(w http.ResponseWriter)) *fakeDynamo {
	f := &fakeDynamo{t: t, replies: replies}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}
		f.calls = append(f.calls, call{Target: r.Header.Get("X-Amz-Target"), Auth: r.Header.Get("Authorization"), Body: body})
		if len(f.calls) > len(f.replies) {
			t.Errorf("unexpected request %d (%s)", len(f.calls), r.Header.Get("X-Amz-Target"))
			w.WriteHeader(http.StatusTeapot)
			return
		}
		f.replies[len(f.calls)-1](w)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeDynamo) store() *DynamoDB {
	d, err := NewDynamoDB(DynamoDBConfig{
		Table: "app-data", Region: "eu-west-2", Endpoint: f.srv.URL,
		AccessKeyID: "AKID", SecretAccessKey: "secret", SessionToken: "tok",
	})
	if err != nil {
		f.t.Fatal(err)
	}
	d.retryBase = time.Millisecond
	return d
}

func reply(status int, body string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

const conditionFailed = `{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException","message":"The conditional request failed"}`

func row(sk, data string, rev int) string {
	b, _ := json.Marshal(data)
	return `{"pk":{"S":"p"},"sk":{"S":"` + sk + `"},"data":{"S":` + string(b) + `},"rev":{"N":"` + itoa(rev) + `"}}`
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func rec(pk, sk, data string) Record {
	return Record{PK: pk, SK: sk, Data: json.RawMessage(data)}
}

func skOf(items []Record) []string {
	out := make([]string, len(items))
	for i, r := range items {
		out[i] = r.SK
	}
	return out
}

func TestNewDynamoDBRefusesUnsafeConfig(t *testing.T) {
	ok := DynamoDBConfig{Table: "app-data", Region: "eu-west-2", AccessKeyID: "a", SecretAccessKey: "s"}
	if _, err := NewDynamoDB(ok); err != nil {
		t.Fatalf("valid config refused: %v", err)
	}
	for name, mutate := range map[string]func(*DynamoDBConfig){
		"no table":              func(c *DynamoDBConfig) { c.Table = "" },
		"table with a slash":    func(c *DynamoDBConfig) { c.Table = "a/b" },
		"short table":           func(c *DynamoDBConfig) { c.Table = "ab" },
		"no region":             func(c *DynamoDBConfig) { c.Region = "" },
		"region as a host":      func(c *DynamoDBConfig) { c.Region = "x.evil.example/" },
		"no access key":         func(c *DynamoDBConfig) { c.AccessKeyID = "" },
		"no secret":             func(c *DynamoDBConfig) { c.SecretAccessKey = "" },
		"remote https endpoint": func(c *DynamoDBConfig) { c.Endpoint = "https://dynamodb.evil.example" },
		"remote http endpoint":  func(c *DynamoDBConfig) { c.Endpoint = "http://example.com:8000" },
		"loopback lookalike":    func(c *DynamoDBConfig) { c.Endpoint = "http://127.0.0.1.evil.example:8000" },
		"userinfo trick":        func(c *DynamoDBConfig) { c.Endpoint = "http://127.0.0.1@evil.example/" },
		"file scheme":           func(c *DynamoDBConfig) { c.Endpoint = "file:///etc/passwd" },
		"endpoint with a path":  func(c *DynamoDBConfig) { c.Endpoint = "http://127.0.0.1:8000/x" },
	} {
		c := ok
		mutate(&c)
		if _, err := NewDynamoDB(c); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	for _, endpoint := range []string{"http://127.0.0.1:8000", "http://localhost:8000", "http://[::1]:8000", "http://localhost"} {
		c := ok
		c.Endpoint = endpoint
		if _, err := NewDynamoDB(c); err != nil {
			t.Errorf("loopback endpoint %s refused: %v", endpoint, err)
		}
	}
}

func TestDynamoDBCreateIsConditionalAndSigned(t *testing.T) {
	f := newFakeDynamo(t, reply(200, `{}`))
	got, err := f.store().Create(context.Background(), rec("p", "s", `{"v":1}`))
	if err != nil || got.Rev != 1 {
		t.Fatalf("Create = %+v, %v", got, err)
	}
	c := f.calls[0]
	if c.Target != "DynamoDB_20120810.PutItem" {
		t.Errorf("target %s", c.Target)
	}
	if c.Body["TableName"] != "app-data" || !strings.Contains(c.Body["ConditionExpression"].(string), "attribute_not_exists") {
		t.Errorf("a create must be a conditional put on the table: %v", c.Body)
	}
	if !strings.Contains(c.Auth, "Credential=AKID/") || !strings.Contains(c.Auth, "/eu-west-2/dynamodb/aws4_request") {
		t.Errorf("request not signed for dynamodb in eu-west-2: %q", c.Auth)
	}
	if strings.Contains(c.Auth, "secret") {
		t.Errorf("secret in the Authorization header")
	}
}

func TestDynamoDBCreateAtTakenKeyIsErrExists(t *testing.T) {
	f := newFakeDynamo(t, reply(400, conditionFailed))
	if _, err := f.store().Create(context.Background(), rec("p", "s", `{}`)); !errors.Is(err, ErrExists) {
		t.Fatalf("Create = %v, want ErrExists", err)
	}
}

func TestDynamoDBReplaceTellsConflictFromMissing(t *testing.T) {
	// Conditional check fails, then the read-back finds a different revision: a conflict.
	f := newFakeDynamo(t, reply(400, conditionFailed), reply(200, `{"Item":`+row("s", `{}`, 4)+`}`))
	if _, err := f.store().Replace(context.Background(), rec("p", "s", `{}`), 3); !errors.Is(err, ErrConflict) {
		t.Fatalf("Replace over a newer record = %v, want ErrConflict", err)
	}
	if got := f.calls[0].Body["ConditionExpression"]; got == nil || !strings.Contains(got.(string), "#rev = :rev") {
		t.Errorf("replace is not guarded by the revision: %v", f.calls[0].Body)
	}

	// Conditional check fails and nothing is there: not found.
	f = newFakeDynamo(t, reply(400, conditionFailed), reply(200, `{}`))
	if _, err := f.store().Replace(context.Background(), rec("p", "s", `{}`), 3); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Replace of nothing = %v, want ErrNotFound", err)
	}
}

func TestDynamoDBDeleteConditionalOnlyWithARevision(t *testing.T) {
	f := newFakeDynamo(t, reply(200, `{}`), reply(200, `{}`))
	d := f.store()
	if err := d.Delete(context.Background(), "p", "s", 0); err != nil {
		t.Fatal(err)
	}
	if _, has := f.calls[0].Body["ConditionExpression"]; has {
		t.Errorf("unconditional delete carries a condition: %v", f.calls[0].Body)
	}
	if err := d.Delete(context.Background(), "p", "s", 2); err != nil {
		t.Fatal(err)
	}
	if _, has := f.calls[1].Body["ConditionExpression"]; !has {
		t.Errorf("revision-checked delete has no condition")
	}
}

func TestDynamoDBGetReadsConsistently(t *testing.T) {
	f := newFakeDynamo(t, reply(200, `{"Item":`+row("s", `{"v":1}`, 2)+`}`), reply(200, `{}`))
	d := f.store()
	got, err := d.Get(context.Background(), "p", "s")
	if err != nil || string(got.Data) != `{"v":1}` || got.Rev != 2 {
		t.Fatalf("Get = %+v, %v", got, err)
	}
	if f.calls[0].Body["ConsistentRead"] != true {
		t.Errorf("Get is not a consistent read: %v", f.calls[0].Body)
	}
	if _, err := d.Get(context.Background(), "p", "none"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get of nothing = %v, want ErrNotFound", err)
	}
}

func TestDynamoDBRefusesMalformedRows(t *testing.T) {
	for name, item := range map[string]string{
		"no data":       `{"pk":{"S":"p"},"sk":{"S":"s"},"rev":{"N":"1"}}`,
		"no rev":        `{"pk":{"S":"p"},"sk":{"S":"s"},"data":{"S":"{}"}}`,
		"bad rev":       `{"pk":{"S":"p"},"sk":{"S":"s"},"data":{"S":"{}"},"rev":{"N":"x"}}`,
		"zero rev":      `{"pk":{"S":"p"},"sk":{"S":"s"},"data":{"S":"{}"},"rev":{"N":"0"}}`,
		"data not set":  `{"pk":{"S":"p"},"sk":{"S":"s"},"data":{"N":"1"},"rev":{"N":"1"}}`,
		"data not json": `{"pk":{"S":"p"},"sk":{"S":"s"},"data":{"S":"{nope"},"rev":{"N":"1"}}`,
	} {
		f := newFakeDynamo(t, reply(200, `{"Item":`+item+`}`))
		if _, err := f.store().Get(context.Background(), "p", "s"); err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("%s: Get = %v, want a malformed-row error", name, err)
		}
	}
}

func TestDynamoDBRetriesThrottlingButNotCallerFaults(t *testing.T) {
	throttle := reply(400, `{"__type":"com.amazonaws.dynamodb.v20120810#ProvisionedThroughputExceededException","message":"slow down"}`)
	f := newFakeDynamo(t, throttle, reply(503, `{}`), reply(200, `{}`))
	if _, err := f.store().Get(context.Background(), "p", "s"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after two retryable failures = %v", err)
	}
	if len(f.calls) != 3 {
		t.Errorf("%d calls, want 3", len(f.calls))
	}

	f = newFakeDynamo(t, throttle, throttle, throttle, throttle, throttle)
	if err := f.store().Delete(context.Background(), "p", "s", 0); err == nil {
		t.Fatal("endless throttling reported success")
	}
	if len(f.calls) != maxAttempts {
		t.Errorf("%d calls, want exactly %d", len(f.calls), maxAttempts)
	}

	f = newFakeDynamo(t, reply(400, `{"__type":"com.amazon.coral.validate#ValidationException","message":"bad"}`))
	if err := f.store().Delete(context.Background(), "p", "s", 0); err == nil || len(f.calls) != 1 {
		t.Errorf("a validation error was retried or swallowed: err %v, %d calls", err, len(f.calls))
	}
}

func TestDynamoDBDoesNotRetryWritesOn5xx(t *testing.T) {
	// A 500 on a write may have applied; a retry would then report the real winner's
	// ErrExists/ErrConflict to the caller whose write took effect. Reads are safe to retry.
	for name, op := range map[string]func(*DynamoDB) error{
		"create":  func(d *DynamoDB) error { _, e := d.Create(context.Background(), rec("p", "s", `{}`)); return e },
		"replace": func(d *DynamoDB) error { _, e := d.Replace(context.Background(), rec("p", "s", `{}`), 1); return e },
		"delete":  func(d *DynamoDB) error { return d.Delete(context.Background(), "p", "s", 0) },
	} {
		f := newFakeDynamo(t, reply(500, `{}`), reply(200, `{}`))
		if err := op(f.store()); err == nil || len(f.calls) != 1 {
			t.Errorf("%s: err %v after %d calls, want the 500 returned after one", name, err, len(f.calls))
		}
	}
	f := newFakeDynamo(t, reply(500, `{}`), reply(200, `{}`))
	if _, err := f.store().Get(context.Background(), "p", "s"); !errors.Is(err, ErrNotFound) || len(f.calls) != 2 {
		t.Errorf("a read was not retried past a 500: %v after %d calls", err, len(f.calls))
	}
	// Throttling means the request was refused, so a write may retry that.
	f = newFakeDynamo(t, reply(400, `{"__type":"x#ThrottlingException"}`), reply(200, `{}`))
	if err := f.store().Delete(context.Background(), "p", "s", 0); err != nil || len(f.calls) != 2 {
		t.Errorf("a throttled write was not retried: %v after %d calls", err, len(f.calls))
	}
}

func TestRevisionOverflowIsInvalid(t *testing.T) {
	f := newFakeDynamo(t)
	if _, err := f.store().Replace(context.Background(), rec("p", "s", `{}`), math.MaxInt64); !errors.Is(err, ErrInvalid) {
		t.Errorf("Replace at MaxInt64 = %v, want ErrInvalid", err)
	}
	if _, err := NewMemory().Replace(context.Background(), rec("p", "s", `{}`), math.MaxInt64); !errors.Is(err, ErrInvalid) {
		t.Errorf("Memory.Replace at MaxInt64 = %v, want ErrInvalid", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("%d requests sent", len(f.calls))
	}
}

func TestDynamoDBClientIgnoresProxyEnvironment(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	d, err := NewDynamoDB(DynamoDBConfig{Table: "app-data", Region: "eu-west-2", AccessKeyID: "a", SecretAccessKey: "s"})
	if err != nil {
		t.Fatal(err)
	}
	tr, ok := d.client.Transport.(*http.Transport)
	if !ok || tr.Proxy != nil {
		t.Fatalf("transport %T must set Proxy nil so signed requests never go via an environment proxy", d.client.Transport)
	}
}

func TestDynamoDBFaultDoesNotLeakTheResponseBody(t *testing.T) {
	f := newFakeDynamo(t, reply(400, `{"__type":"x#AccessDeniedException","message":"User: arn:aws:sts::123456789012:assumed-role/secret-role is not authorized"}`))
	err := f.store().Delete(context.Background(), "p", "s", 0)
	if err == nil || !strings.Contains(err.Error(), "AccessDeniedException") {
		t.Fatalf("err = %v, want the exception type", err)
	}
	if strings.Contains(err.Error(), "123456789012") {
		t.Errorf("the error carries account detail: %v", err)
	}
}

func TestDynamoDBStopsOnContextCancel(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		reply(503, `{}`)(w)
	}))
	defer srv.Close()
	d, err := NewDynamoDB(DynamoDBConfig{Table: "app-data", Region: "eu-west-2", Endpoint: srv.URL, AccessKeyID: "a", SecretAccessKey: "s"})
	if err != nil {
		t.Fatal(err)
	}
	d.retryBase = time.Hour // a retry would hang the test unless the context ends it
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := d.Delete(ctx, "p", "s", 0); err == nil {
		t.Fatal("no error")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("the retry wait ignored the context")
	}
}

func TestDynamoDBDoesNotFollowRedirects(t *testing.T) {
	var followed atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { followed.Store(true) }))
	defer target.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer srv.Close()
	d, _ := NewDynamoDB(DynamoDBConfig{Table: "app-data", Region: "eu-west-2", Endpoint: srv.URL, AccessKeyID: "a", SecretAccessKey: "s"})
	if err := d.Delete(context.Background(), "p", "s", 0); err == nil {
		t.Error("a redirect was treated as success")
	}
	if followed.Load() {
		t.Error("a signed request was replayed to a redirect target")
	}
}

func TestDynamoDBCapsTheResponseBody(t *testing.T) {
	f := newFakeDynamo(t, func(w http.ResponseWriter) {
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{"Item":{"data":{"S":"`)
		chunk := strings.Repeat("a", 1<<20)
		for i := 0; i < maxResponseBytes>>20+2; i++ {
			_, _ = io.WriteString(w, chunk)
		}
	})
	if _, err := f.store().Get(context.Background(), "p", "s"); err == nil {
		t.Fatal("an oversize response was accepted")
	}
}

func TestDynamoDBQueryFillsAPageAcrossTruncatedResponses(t *testing.T) {
	// DynamoDB stops at 1 MB and hands back a LastEvaluatedKey with fewer rows than asked
	// for. The page must keep reading until it has Limit+1 rows or the partition ends, so
	// NextCursor means "there is more" and a short page means "that was all".
	f := newFakeDynamo(t,
		reply(200, `{"Items":[`+row("a", `{}`, 1)+`],"LastEvaluatedKey":{"pk":{"S":"p"},"sk":{"S":"a"}}}`),
		reply(200, `{"Items":[`+row("b", `{}`, 1)+`,`+row("c", `{}`, 1)+`]}`),
	)
	page, err := f.store().Query(context.Background(), Query{PK: "p", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(skOf(page.Items), ","); got != "a,b" || page.NextCursor == "" {
		t.Fatalf("page %s cursor %q, want a,b and a cursor", got, page.NextCursor)
	}
	if esk := f.calls[1].Body["ExclusiveStartKey"]; esk == nil {
		t.Errorf("the second read did not resume from LastEvaluatedKey")
	}
	if f.calls[0].Body["Limit"] != float64(3) || f.calls[0].Body["ConsistentRead"] != true {
		t.Errorf("first read: %v", f.calls[0].Body)
	}
	if f.calls[0].Body["ScanIndexForward"] != true {
		t.Errorf("a forward query must scan forward: %v", f.calls[0].Body)
	}
}

func TestDynamoDBQueryNeverStopsOnAShortTruncatedPage(t *testing.T) {
	f := newFakeDynamo(t,
		reply(200, `{"Items":[`+row("a", `{}`, 1)+`],"LastEvaluatedKey":{"pk":{"S":"p"},"sk":{"S":"a"}}}`),
		reply(200, `{"Items":[]}`),
	)
	page, err := f.store().Query(context.Background(), Query{PK: "p", Limit: 2})
	if err != nil || len(page.Items) != 1 || page.NextCursor != "" {
		t.Fatalf("page %+v err %v, want one record and no cursor", page, err)
	}
}

func TestDynamoDBQueryUsesTheKeyConditionNotAFilter(t *testing.T) {
	f := newFakeDynamo(t, reply(200, `{"Items":[]}`))
	q := Query{PK: "p", SKPrefix: "inv#", Reverse: true, Limit: 5}
	q.Cursor = encodeCursor("p", "inv#009")
	if _, err := f.store().Query(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	b := f.calls[0].Body
	if !strings.Contains(b["KeyConditionExpression"].(string), "begins_with") || b["FilterExpression"] != nil {
		t.Errorf("prefix must be a key condition: %v", b)
	}
	if b["ScanIndexForward"] != false {
		t.Errorf("reverse query scans forward: %v", b)
	}
	esk, _ := b["ExclusiveStartKey"].(map[string]any)
	if esk == nil || esk["sk"].(map[string]any)["S"] != "inv#009" || esk["pk"].(map[string]any)["S"] != "p" {
		t.Errorf("cursor did not become ExclusiveStartKey: %v", b["ExclusiveStartKey"])
	}
	// Values are bound as attribute values, never spliced into an expression.
	if strings.Contains(b["KeyConditionExpression"].(string), "inv#") {
		t.Errorf("a prefix was spliced into the expression: %v", b["KeyConditionExpression"])
	}
}

func TestDynamoDBRejectsInvalidInputBeforeTheNetwork(t *testing.T) {
	f := newFakeDynamo(t) // any request is an unexpected-call failure
	d := f.store()
	if _, err := d.Create(context.Background(), rec("", "s", `{}`)); !errors.Is(err, ErrInvalid) {
		t.Errorf("Create = %v", err)
	}
	if _, err := d.Query(context.Background(), Query{PK: "p", Limit: 0}); !errors.Is(err, ErrInvalid) {
		t.Errorf("Query = %v", err)
	}
	if _, err := d.Query(context.Background(), Query{PK: "p", Limit: 1, Cursor: "forged"}); !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("Query = %v", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("%d requests sent for invalid input", len(f.calls))
	}
}
