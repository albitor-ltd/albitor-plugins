package apikit

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Every list endpoint is paginated. An unbounded list is a latency and cost
// defect that only shows up once real data exists: the page that scanned five
// records in development scans the whole partition in production, and past the
// response size limit it truncates rather than failing.
const (
	// DefaultPageLimit applies when the caller sends no limit.
	DefaultPageLimit = 20
	// MaxPageLimit is a hard ceiling. A larger limit is rejected rather than
	// clamped: silently returning fewer records than asked for reads to the
	// caller as "that was the last page".
	MaxPageLimit = 100
	// MaxSortKeyBytes bounds the sort key a cursor may name. DynamoDB's limit
	// on a sort key is 1024 bytes, and a longer key cannot be a real one.
	MaxSortKeyBytes = 1024
)

// cursorPrefix versions the opaque cursor. The cursor is a server-owned token;
// the prefix means a cursor minted by an older encoding is rejected outright
// instead of being misread as a key.
const cursorPrefix = "v1:"

var (
	// ErrInvalidLimit is what ParseLimit returns for a limit outside 1 to
	// MaxPageLimit.
	ErrInvalidLimit = errors.New("limit must be an integer between 1 and " + strconv.Itoa(MaxPageLimit))
	// ErrInvalidCursor is the one cursor error a data source returns; ServeList
	// answers it with a 400.
	ErrInvalidCursor = errors.New("cursor is not a cursor this server issued")
)

// Page is the list envelope. Every list endpoint returns this shape, never a
// bare array: a bare array has nowhere to put a cursor, so adding pagination to
// one later is a breaking change for every client.
//
// NextCursor is absent on the last page. A present cursor always means there is
// at least one more record.
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// PageQuery is one request for a page: the opaque cursor the previous page
// returned ("" for the first) and a limit already validated by ParseLimit.
type PageQuery struct {
	Cursor string
	Limit  int
}

// ParseLimit reads the limit query parameter. An absent limit takes the
// default; anything unparseable, non-positive or above the ceiling is a client
// error rather than a silent correction.
func ParseLimit(query url.Values) (int, error) {
	raw := strings.TrimSpace(query.Get("limit"))
	if raw == "" {
		return DefaultPageLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > MaxPageLimit {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}

// EncodeCursor mints an opaque cursor naming the list it belongs to and the
// sort key of the last record on a page. It is opaque on purpose: clients must
// round-trip it, never construct or decompose it, so its encoding stays a
// server implementation detail.
func EncodeCursor(collection, key string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(cursorPrefix + collection + ":" + key))
}

// DecodeCursor recovers the sort key to resume after. An empty cursor means
// "start at the beginning" and is not an error. A cursor minted for another
// list is refused: its key means nothing in this one.
func DecodeCursor(collection, cursor string) (string, error) {
	if cursor == "" {
		return "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", ErrInvalidCursor
	}
	rest, ok := strings.CutPrefix(string(raw), cursorPrefix+collection+":")
	if !ok || rest == "" || len(rest) > MaxSortKeyBytes {
		return "", ErrInvalidCursor
	}
	return rest, nil
}

// ServeList answers one page of a collection. Every list endpoint goes through
// it, so no endpoint can quietly ship without a limit and a cursor. list
// returns ErrInvalidCursor (from DecodeCursor) for a cursor it did not issue;
// any other error is a storage fault, logged against the correlation id and
// answered as an opaque 500, never as a quietly short page.
func ServeList[T any](w http.ResponseWriter, r *http.Request, logger *slog.Logger, list func(PageQuery) (Page[T], error)) {
	query := r.URL.Query()

	limit, err := ParseLimit(query)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	page, err := list(PageQuery{Cursor: strings.TrimSpace(query.Get("cursor")), Limit: limit})
	if err != nil {
		if errors.Is(err, ErrInvalidCursor) {
			WriteError(w, http.StatusBadRequest, "invalid cursor")
			return
		}
		WriteFailure(w, r, logger, "list records", err)
		return
	}
	if page.Items == nil {
		page.Items = []T{} // an empty page is [], never null
	}
	WriteJSON(w, http.StatusOK, page)
}

// WriteFailure logs the cause against the request's correlation id and answers
// an opaque 500. The detail belongs in the log, not in the response.
func WriteFailure(w http.ResponseWriter, r *http.Request, logger *slog.Logger, op string, err error) {
	logger.ErrorContext(r.Context(), "operation failed",
		slog.String("correlation_id", CorrelationIDFromContext(r.Context())),
		slog.String("operation", op),
		slog.String("path", r.URL.Path),
		slog.String("error", err.Error()),
	)
	WriteError(w, http.StatusInternalServerError, "internal error")
}
