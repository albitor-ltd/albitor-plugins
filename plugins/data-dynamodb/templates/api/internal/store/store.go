// Package store is the data-store@1 seam: the one interface every handler uses
// to read and write the app's records, with two adapters that must behave
// identically (memory.go for a local run and tests, dynamodb.go deployed).
//
// THE TABLE. One table with a string partition key `pk` and a string sort key
// `sk`; every row carries the record's JSON in `data` and a revision in `rev`.
// A handler never opens a database client: a record kind is a function over
// Store, so the local run, where the app's specs are verified, exercises the
// same code that runs deployed.
//
// THE KEYS. Build them with Key, never by concatenating request input: Key
// refuses the separator and control characters, so a part taken from a request
// cannot name a key outside the one the caller built. The owner of a record
// comes from the verified identity, never the request body.
//
// This is platform code, maintained with the plugin that ships it; extend the
// app around it rather than editing it. The conformance suite in storetest
// runs against both adapters, so a behaviour a handler relies on is one the
// deployed store has.
package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"webapp/internal/apikit"
)

const (
	// MaxKeyBytes bounds a partition key, a sort key and a sort-key prefix.
	// DynamoDB allows 1024 bytes of sort key; the same ceiling on every key
	// keeps the adapters equal.
	MaxKeyBytes = 1024
	// MaxDataBytes bounds one record's JSON. DynamoDB's item limit is 400 KiB;
	// the margin leaves room for the keys and the revision.
	MaxDataBytes = 256 << 10
	// KeySeparator joins the parts of a key built by Key.
	KeySeparator = "#"
)

var (
	// ErrNotFound means no record has the key (or, for a conditional write, the
	// record the caller meant to change is gone).
	ErrNotFound = errors.New("record not found")
	// ErrExists means Create found a record already at the key.
	ErrExists = errors.New("record already exists")
	// ErrConflict means the record changed since the caller read it: the
	// expected revision is not the stored one. Re-read and retry.
	ErrConflict = errors.New("record changed concurrently")
	// ErrInvalid is a rejected argument: a bad key, oversized or non-JSON data,
	// a limit out of range. A caller fault, never a storage fault.
	ErrInvalid = errors.New("invalid argument")
	// ErrInvalidCursor is apikit's, so apikit.ServeList answers it with a 400.
	ErrInvalidCursor = apikit.ErrInvalidCursor
)

// Record is one stored row. Data is the record's JSON, opaque to the store.
// Rev is 0 on a record not yet stored and 1 or more on a stored one; the store
// owns it, and Replace takes the revision the caller read.
type Record struct {
	PK   string
	SK   string
	Data json.RawMessage
	Rev  int64
}

// Query is one list request over a single partition.
type Query struct {
	// PK is the partition, exact. Required.
	PK string
	// SKPrefix narrows the page to sort keys that start with it. Empty is all.
	SKPrefix string
	// Reverse lists in descending sort-key order.
	Reverse bool
	// Cursor is the opaque NextCursor of the previous page, "" for the first.
	Cursor string
	// Limit is 1 to apikit.MaxPageLimit.
	Limit int
}

// Store is the data layer. Implementations are safe for concurrent use.
type Store interface {
	// Get returns the record at the key, or ErrNotFound.
	Get(ctx context.Context, pk, sk string) (Record, error)
	// Create stores a new record at revision 1, or ErrExists. r.Rev is ignored.
	Create(ctx context.Context, r Record) (Record, error)
	// Replace overwrites the record at r's key if its revision is
	// expectedRev (1 or more), storing it at expectedRev+1. ErrNotFound when
	// there is no record, ErrConflict when the revision has moved.
	Replace(ctx context.Context, r Record, expectedRev int64) (Record, error)
	// Delete removes the record. With expectedRev 0 it is unconditional and
	// idempotent: no record is not an error. With expectedRev 1 or more it
	// deletes only that revision: ErrNotFound or ErrConflict otherwise.
	Delete(ctx context.Context, pk, sk string, expectedRev int64) error
	// Query answers one page of a partition in sort-key order. NextCursor is
	// set only when at least one more record follows.
	Query(ctx context.Context, q Query) (apikit.Page[Record], error)
}

// Key joins parts into a partition or sort key. A part must be non-empty, free
// of the separator and of control characters, and valid UTF-8:
//
//	pk, err := store.Key("OWNER", owner, "ITEM") // OWNER#<owner>#ITEM
func Key(parts ...string) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("%w: a key needs at least one part", ErrInvalid)
	}
	for i, p := range parts {
		if err := checkText(p, "key part "+strconv.Itoa(i), false); err != nil {
			return "", err
		}
		if strings.Contains(p, KeySeparator) {
			return "", fmt.Errorf("%w: key part %d contains %q", ErrInvalid, i, KeySeparator)
		}
	}
	key := strings.Join(parts, KeySeparator)
	if len(key) > MaxKeyBytes {
		return "", fmt.Errorf("%w: key is longer than %d bytes", ErrInvalid, MaxKeyBytes)
	}
	return key, nil
}

// checkText validates a key, part or prefix. Empty is allowed only where
// allowEmpty says so.
func checkText(s, what string, allowEmpty bool) error {
	switch {
	case s == "":
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("%w: %s is empty", ErrInvalid, what)
	case len(s) > MaxKeyBytes:
		return fmt.Errorf("%w: %s is longer than %d bytes", ErrInvalid, what, MaxKeyBytes)
	case !utf8.ValidString(s):
		return fmt.Errorf("%w: %s is not valid UTF-8", ErrInvalid, what)
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w: %s contains a control character", ErrInvalid, what)
		}
	}
	return nil
}

func checkKey(pk, sk string) error {
	if err := checkText(pk, "partition key", false); err != nil {
		return err
	}
	return checkText(sk, "sort key", false)
}

func checkData(data json.RawMessage) error {
	if len(data) == 0 {
		return fmt.Errorf("%w: data is empty", ErrInvalid)
	}
	if len(data) > MaxDataBytes {
		return fmt.Errorf("%w: data is longer than %d bytes", ErrInvalid, MaxDataBytes)
	}
	if !json.Valid(data) {
		return fmt.Errorf("%w: data is not valid JSON", ErrInvalid)
	}
	return nil
}

func checkWrite(r Record) error {
	if err := checkKey(r.PK, r.SK); err != nil {
		return err
	}
	return checkData(r.Data)
}

func checkRev(rev int64, zeroOK bool) error {
	if rev < 0 || (rev == 0 && !zeroOK) {
		return fmt.Errorf("%w: revision %d", ErrInvalid, rev)
	}
	return nil
}

func checkQuery(q Query) error {
	if err := checkText(q.PK, "partition key", false); err != nil {
		return err
	}
	if err := checkText(q.SKPrefix, "sort-key prefix", true); err != nil {
		return err
	}
	if q.Limit < 1 || q.Limit > apikit.MaxPageLimit {
		return fmt.Errorf("%w: limit %d outside 1 to %d", ErrInvalid, q.Limit, apikit.MaxPageLimit)
	}
	return nil
}

// cursorPrefix versions the cursor, as apikit's does.
const cursorPrefix = "v1:"

// encodeCursor names the partition and the sort key of the last record on a
// page. It is opaque: callers round-trip it and never read it.
func encodeCursor(pk, lastSK string) string {
	raw := cursorPrefix + strconv.Itoa(len(pk)) + ":" + pk + lastSK
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeCursor recovers the sort key to resume after. A cursor minted for
// another partition or prefix is refused: its key means nothing here.
func decodeCursor(q Query) (string, error) {
	if q.Cursor == "" {
		return "", nil
	}
	if len(q.Cursor) > base64.RawURLEncoding.EncodedLen(len(cursorPrefix)+8+2*MaxKeyBytes) {
		return "", ErrInvalidCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(q.Cursor)
	if err != nil {
		return "", ErrInvalidCursor
	}
	rest, ok := strings.CutPrefix(string(raw), cursorPrefix)
	if !ok {
		return "", ErrInvalidCursor
	}
	n, rest, ok := strings.Cut(rest, ":")
	if !ok {
		return "", ErrInvalidCursor
	}
	size, err := strconv.Atoi(n)
	if err != nil || size < 1 || size > len(rest) || strconv.Itoa(size) != n {
		return "", ErrInvalidCursor
	}
	pk, sk := rest[:size], rest[size:]
	if pk != q.PK || sk == "" || len(sk) > MaxKeyBytes || !strings.HasPrefix(sk, q.SKPrefix) || !utf8.ValidString(sk) {
		return "", ErrInvalidCursor
	}
	return sk, nil
}

// clone copies a record so a caller can never alias an adapter's storage.
func clone(r Record) Record {
	r.Data = append(json.RawMessage(nil), r.Data...)
	return r
}
