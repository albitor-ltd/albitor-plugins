package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"webapp/internal/apikit"
)

const (
	// Attribute names on every row.
	attrPK   = "pk"
	attrSK   = "sk"
	attrData = "data"
	// attrRev is the record's revision, the condition that keeps two writers
	// from both applying against the same prior value.
	attrRev = "rev"

	// maxAttempts bounds one call, retries included.
	maxAttempts = 4
	// maxResponseBytes bounds what is read from DynamoDB. A single query
	// response is at most 1 MB of items; the margin covers its JSON framing.
	maxResponseBytes = 4 << 20
	// maxQueryReads bounds the reads one page may take when DynamoDB cuts its
	// responses short at 1 MB.
	maxQueryReads = 25
	// requestTimeout bounds one HTTP exchange.
	requestTimeout = 10 * time.Second
)

var (
	regionRE = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-[0-9]$`)
	tableRE  = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,255}$`)
)

// DynamoDBConfig configures the deployed adapter.
type DynamoDBConfig struct {
	// Table is the AWS table name the platform gave the function (see Open).
	Table  string
	Region string
	// Endpoint overrides the regional endpoint for DynamoDB Local. It must be
	// an http(s) URL naming a loopback host and nothing else: a signed
	// request must never be redirected to a host the platform did not name.
	Endpoint string

	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// DynamoDB is the adapter a deployed app runs on: one table keyed pk and sk.
// It speaks DynamoDB's JSON API with the standard library, signed with SigV4,
// so the generated module needs no SDK.
type DynamoDB struct {
	table    string
	region   string
	endpoint string
	creds    awsCredentials
	client   *http.Client
	now      func() time.Time
	// retryBase is the first retry delay; it doubles with jitter. A field so
	// tests do not sleep.
	retryBase time.Duration
}

var _ Store = (*DynamoDB)(nil)

// NewDynamoDB validates cfg and returns the adapter. It makes no call.
func NewDynamoDB(cfg DynamoDBConfig) (*DynamoDB, error) {
	switch {
	case !tableRE.MatchString(cfg.Table):
		return nil, errors.New("dynamodb: the table name is missing or not a table name")
	case !regionRE.MatchString(cfg.Region):
		return nil, errors.New("dynamodb: the region is missing or not a region")
	case cfg.AccessKeyID == "" || cfg.SecretAccessKey == "":
		return nil, errors.New("dynamodb: no AWS credentials")
	}
	endpoint := "https://dynamodb." + cfg.Region + ".amazonaws.com"
	if cfg.Endpoint != "" {
		var err error
		if endpoint, err = loopbackEndpoint(cfg.Endpoint); err != nil {
			return nil, err
		}
	}
	return &DynamoDB{
		table:    cfg.Table,
		region:   cfg.Region,
		endpoint: endpoint,
		creds:    awsCredentials{cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken},
		client: &http.Client{
			Timeout: requestTimeout,
			// No proxy from the environment: a proxy would see the signed request and the
			// session token, and a poisoned HTTPS_PROXY would capture them.
			Transport: &http.Transport{Proxy: nil, ForceAttemptHTTP2: true, MaxIdleConns: 10, IdleConnTimeout: 90 * time.Second, TLSHandshakeTimeout: 10 * time.Second},
			// A signed request is never replayed elsewhere.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		now:       time.Now,
		retryBase: 50 * time.Millisecond,
	}, nil
}

// loopbackEndpoint accepts only scheme://loopback-host[:port], so a poisoned
// environment cannot point signed requests at another host.
func loopbackEndpoint(raw string) (string, error) {
	u, err := url.Parse(raw)
	bad := errors.New("dynamodb: the endpoint override must be http(s)://localhost, 127.0.0.1 or [::1], with an optional port")
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return "", bad
	}
	host := u.Hostname()
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return "", bad
		}
	}
	return u.Scheme + "://" + u.Host, nil
}

// attr is a DynamoDB attribute value, limited to the two types a row uses.
type attr struct {
	S *string `json:"S,omitempty"`
	N *string `json:"N,omitempty"`
}

func strAttr(s string) attr { return attr{S: &s} }
func numAttr(n int64) attr  { s := strconv.FormatInt(n, 10); return attr{N: &s} }

type item map[string]attr

func rowItem(r Record, rev int64) item {
	return item{
		attrPK:   strAttr(r.PK),
		attrSK:   strAttr(r.SK),
		attrData: strAttr(string(r.Data)),
		attrRev:  numAttr(rev),
	}
}

func keyItem(pk, sk string) item {
	return item{attrPK: strAttr(pk), attrSK: strAttr(sk)}
}

// record decodes a stored row, refusing one this adapter did not write.
func (it item) record() (Record, error) {
	var r Record
	var ok bool
	if r.PK, ok = it.str(attrPK); !ok {
		return Record{}, errMalformed
	}
	if r.SK, ok = it.str(attrSK); !ok {
		return Record{}, errMalformed
	}
	data, ok := it.str(attrData)
	if !ok || checkData(json.RawMessage(data)) != nil {
		return Record{}, errMalformed
	}
	r.Data = json.RawMessage(data)
	rev, ok := it[attrRev]
	if !ok || rev.N == nil {
		return Record{}, errMalformed
	}
	n, err := strconv.ParseInt(*rev.N, 10, 64)
	if err != nil || n < 1 {
		return Record{}, errMalformed
	}
	r.Rev = n
	return r, nil
}

func (it item) str(name string) (string, bool) {
	a, ok := it[name]
	if !ok || a.S == nil || !utf8.ValidString(*a.S) {
		return "", false
	}
	return *a.S, true
}

var errMalformed = errors.New("dynamodb: a stored row is not one this store wrote")

// Get implements Store.
func (d *DynamoDB) Get(ctx context.Context, pk, sk string) (Record, error) {
	if err := checkKey(pk, sk); err != nil {
		return Record{}, err
	}
	var out struct{ Item item }
	err := d.call(ctx, "GetItem", map[string]any{
		"TableName": d.table, "Key": keyItem(pk, sk), "ConsistentRead": true,
	}, &out)
	if err != nil {
		return Record{}, fmt.Errorf("get: %w", err)
	}
	if out.Item == nil {
		return Record{}, ErrNotFound
	}
	return out.Item.record()
}

// Create implements Store.
func (d *DynamoDB) Create(ctx context.Context, r Record) (Record, error) {
	if err := checkWrite(r); err != nil {
		return Record{}, err
	}
	err := d.call(ctx, "PutItem", map[string]any{
		"TableName":                d.table,
		"Item":                     rowItem(r, 1),
		"ConditionExpression":      "attribute_not_exists(#pk)",
		"ExpressionAttributeNames": map[string]string{"#pk": attrPK},
	}, nil)
	if isType(err, "ConditionalCheckFailedException") {
		return Record{}, ErrExists
	}
	if err != nil {
		return Record{}, fmt.Errorf("create: %w", err)
	}
	r.Rev = 1
	return clone(r), nil
}

// Replace implements Store.
func (d *DynamoDB) Replace(ctx context.Context, r Record, expectedRev int64) (Record, error) {
	if err := checkWrite(r); err != nil {
		return Record{}, err
	}
	if err := checkRev(expectedRev, false); err != nil {
		return Record{}, err
	}
	err := d.call(ctx, "PutItem", map[string]any{
		"TableName":                 d.table,
		"Item":                      rowItem(r, expectedRev+1),
		"ConditionExpression":       "#rev = :rev",
		"ExpressionAttributeNames":  map[string]string{"#rev": attrRev},
		"ExpressionAttributeValues": map[string]attr{":rev": numAttr(expectedRev)},
	}, nil)
	if isType(err, "ConditionalCheckFailedException") {
		return Record{}, d.whyRefused(ctx, r.PK, r.SK)
	}
	if err != nil {
		return Record{}, fmt.Errorf("replace: %w", err)
	}
	r.Rev = expectedRev + 1
	return clone(r), nil
}

// Delete implements Store.
func (d *DynamoDB) Delete(ctx context.Context, pk, sk string, expectedRev int64) error {
	if err := checkKey(pk, sk); err != nil {
		return err
	}
	if err := checkRev(expectedRev, true); err != nil {
		return err
	}
	in := map[string]any{"TableName": d.table, "Key": keyItem(pk, sk)}
	if expectedRev > 0 {
		in["ConditionExpression"] = "#rev = :rev"
		in["ExpressionAttributeNames"] = map[string]string{"#rev": attrRev}
		in["ExpressionAttributeValues"] = map[string]attr{":rev": numAttr(expectedRev)}
	}
	err := d.call(ctx, "DeleteItem", in, nil)
	if isType(err, "ConditionalCheckFailedException") {
		return d.whyRefused(ctx, pk, sk)
	}
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// whyRefused turns a failed revision condition into ErrNotFound (nothing is
// there) or ErrConflict (something is, at another revision).
func (d *DynamoDB) whyRefused(ctx context.Context, pk, sk string) error {
	if _, err := d.Get(ctx, pk, sk); errors.Is(err, ErrNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return ErrConflict
}

// Query implements Store. It reads Limit+1 rows so that a cursor is issued
// only when a record follows the page, and keeps reading when DynamoDB cuts a
// response short at 1 MB.
func (d *DynamoDB) Query(ctx context.Context, q Query) (apikit.Page[Record], error) {
	empty := apikit.Page[Record]{Items: []Record{}}
	if err := checkQuery(q); err != nil {
		return empty, err
	}
	after, err := decodeCursor(q)
	if err != nil {
		return empty, err
	}

	cond := "#pk = :pk"
	names := map[string]string{"#pk": attrPK}
	values := map[string]attr{":pk": strAttr(q.PK)}
	if q.SKPrefix != "" {
		cond += " AND begins_with(#sk, :prefix)"
		names["#sk"] = attrSK
		values[":prefix"] = strAttr(q.SKPrefix)
	}

	want := q.Limit + 1
	var rows []Record
	var start item
	if after != "" {
		start = keyItem(q.PK, after)
	}
	for reads := 0; len(rows) < want; reads++ {
		if reads == maxQueryReads {
			return empty, errors.New("query: DynamoDB cut the page short too many times")
		}
		in := map[string]any{
			"TableName":                 d.table,
			"KeyConditionExpression":    cond,
			"ExpressionAttributeNames":  names,
			"ExpressionAttributeValues": values,
			"ScanIndexForward":          !q.Reverse,
			"ConsistentRead":            true,
			"Limit":                     want - len(rows),
		}
		if start != nil {
			in["ExclusiveStartKey"] = start
		}
		var out struct {
			Items            []item
			LastEvaluatedKey item
		}
		if err := d.call(ctx, "Query", in, &out); err != nil {
			return empty, fmt.Errorf("query: %w", err)
		}
		for _, it := range out.Items {
			r, err := it.record()
			if err != nil {
				return empty, err
			}
			if r.PK != q.PK {
				return empty, errMalformed
			}
			rows = append(rows, r)
		}
		if out.LastEvaluatedKey == nil {
			break
		}
		start = out.LastEvaluatedKey
	}
	if len(rows) > want {
		rows = rows[:want]
	}
	return pageOf(q, rows), nil
}

// apiError is a DynamoDB error response. Only the exception type and status
// are kept: the message can name the caller's role and account, and this error
// reaches a log.
type apiError struct {
	Op     string
	Status int
	Type   string
}

func (e *apiError) Error() string {
	t := e.Type
	if t == "" {
		t = "unrecognised response"
	}
	return fmt.Sprintf("dynamodb %s: %s (HTTP %d)", e.Op, t, e.Status)
}

func isType(err error, typ string) bool {
	var ae *apiError
	return errors.As(err, &ae) && ae.Type == typ
}

// retryable reports whether a failed call may be sent again. Throttling means
// the request was refused, so any call may retry it. A 5xx says nothing about
// whether a write applied, so only a read retries it: retrying a write whose
// first attempt did apply would answer ErrExists or ErrConflict, blaming the
// real winner on someone else.
func retryable(e *apiError, read bool) bool {
	switch e.Type {
	case "ThrottlingException", "ProvisionedThroughputExceededException", "RequestLimitExceeded":
		return true
	}
	return read && e.Status >= 500
}

// call sends one signed DynamoDB request and decodes the reply into out (nil to
// discard it). It retries throttling, and 5xx on reads only (see retryable). A
// network failure is never retried, for the same reason as a write's 5xx.
func (d *DynamoDB) call(ctx context.Context, op string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	var last error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			if err := d.wait(ctx, attempt); err != nil {
				return err
			}
		}
		status, payload, err := d.send(ctx, op, body)
		if err != nil {
			return err
		}
		if status == http.StatusOK {
			if out == nil {
				return nil
			}
			if err := json.Unmarshal(payload, out); err != nil {
				return fmt.Errorf("dynamodb %s: unreadable response", op)
			}
			return nil
		}
		ae := &apiError{Op: op, Status: status}
		var e struct {
			Type string `json:"__type"`
		}
		if json.Unmarshal(payload, &e) == nil {
			_, ae.Type, _ = strings.Cut(e.Type, "#")
			if ae.Type == "" {
				ae.Type = e.Type
			}
		}
		if !retryable(ae, op == "GetItem" || op == "Query") {
			return ae
		}
		last = ae
	}
	return last
}

func (d *DynamoDB) send(ctx context.Context, op string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810."+op)
	signRequest(req, body, d.creds, d.region, "dynamodb", d.now())

	resp, err := d.client.Do(req)
	if err != nil {
		// The url.Error would echo the endpoint; the cause is enough.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return 0, nil, fmt.Errorf("dynamodb %s: %w", op, err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return 0, nil, fmt.Errorf("dynamodb %s: reading the response: %w", op, err)
	}
	if len(payload) > maxResponseBytes {
		return 0, nil, fmt.Errorf("dynamodb %s: the response is larger than %d bytes", op, maxResponseBytes)
	}
	return resp.StatusCode, payload, nil
}

// wait sleeps the jittered exponential delay before retry number attempt, and
// returns early with the context's error if it ends.
func (d *DynamoDB) wait(ctx context.Context, attempt int) error {
	delay := d.retryBase << (attempt - 1)
	delay = delay/2 + time.Duration(rand.Int64N(int64(delay)/2+1))
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
