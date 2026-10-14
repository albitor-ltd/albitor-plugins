package apikit

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

const testCollection = "widget"

func quietLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// widgets serves n records, "w001" upwards, through ServeList: the same shape
// an app's list handler has over its store, with cursors that name a sort key.
func widgets(n int) http.Handler {
	key := func(i int) string { return "w" + strconv.Itoa(1000 + i)[1:] }
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeList(w, r, quietLogger(), func(q PageQuery) (Page[string], error) {
			after, err := DecodeCursor(testCollection, q.Cursor)
			if err != nil {
				return Page[string]{}, err
			}
			var items []string
			next := ""
			for i := 1; i <= n; i++ {
				if key(i) <= after {
					continue
				}
				if len(items) == q.Limit {
					next = EncodeCursor(testCollection, items[len(items)-1])
					break
				}
				items = append(items, key(i))
			}
			return Page[string]{Items: items, NextCursor: next}, nil
		})
	})
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func decodePage(t *testing.T, rec *httptest.ResponseRecorder) Page[string] {
	t.Helper()
	var p Page[string]
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("body is not a page (%q): %v", rec.Body.String(), err)
	}
	return p
}

// TestListPagesThroughEveryRecordExactlyOnce fails if pagination is absent: an
// unpaginated list returns all 45 records on the first response and never
// mints a cursor.
func TestListPagesThroughEveryRecordExactlyOnce(t *testing.T) {
	const seeded = 45 // well over DefaultPageLimit, and not a multiple of it
	h := widgets(seeded)

	seen := map[string]int{}
	cursor, pages := "", 0
	for {
		target := "/"
		if cursor != "" {
			target += "?cursor=" + url.QueryEscape(cursor)
		}
		rec := get(t, h, target)
		if rec.Code != http.StatusOK {
			t.Fatalf("page %d: status = %d (%s)", pages+1, rec.Code, rec.Body.String())
		}
		page := decodePage(t, rec)
		pages++
		if len(page.Items) == 0 {
			t.Fatalf("page %d is empty; a cursor was emitted with nothing left", pages)
		}
		if pages == 1 && len(page.Items) != DefaultPageLimit {
			t.Fatalf("first page has %d items, want %d", len(page.Items), DefaultPageLimit)
		}
		for _, id := range page.Items {
			seen[id]++
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
		if pages > seeded {
			t.Fatalf("paging did not terminate after %d pages", pages)
		}
	}
	if pages != 3 {
		t.Fatalf("pages = %d, want 3 for %d records at a limit of %d", pages, seeded, DefaultPageLimit)
	}
	if len(seen) != seeded {
		t.Fatalf("distinct records = %d, want %d", len(seen), seeded)
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("record %q returned %d times", id, n)
		}
	}
}

func TestLastPageCarriesNoCursor(t *testing.T) {
	// Exactly one full page: a cursor here would promise a page that is empty.
	page := decodePage(t, get(t, widgets(DefaultPageLimit), "/"))
	if len(page.Items) != DefaultPageLimit || page.NextCursor != "" {
		t.Fatalf("items = %d, cursor = %q; want a full last page with no cursor", len(page.Items), page.NextCursor)
	}
}

func TestEmptyListIsAnEmptyArrayNotNull(t *testing.T) {
	rec := get(t, widgets(0), "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"items":[]}` {
		t.Fatalf("body = %s, want {\"items\":[]}", got)
	}
}

func TestLimitIsHonoured(t *testing.T) {
	page := decodePage(t, get(t, widgets(12), "/?limit=5"))
	if len(page.Items) != 5 || page.NextCursor == "" {
		t.Fatalf("items = %d, cursor = %q; want 5 and a cursor", len(page.Items), page.NextCursor)
	}
}

func TestLimitOutOfRangeIsRejectedNotClamped(t *testing.T) {
	for _, limit := range []string{"0", "-1", "abc", "1.5", strconv.Itoa(MaxPageLimit + 1)} {
		t.Run("limit="+limit, func(t *testing.T) {
			rec := get(t, widgets(3), "/?limit="+url.QueryEscape(limit))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
	if rec := get(t, widgets(3), "/?limit="+strconv.Itoa(MaxPageLimit)); rec.Code != http.StatusOK {
		t.Fatalf("limit = MaxPageLimit: status = %d, want 200", rec.Code)
	}
}

func TestACursorTheServerDidNotIssueIsRejected(t *testing.T) {
	prefixed := func(prefix, rest string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(prefix + rest))
	}
	cases := map[string]string{
		"not base64":       "!!!!",
		"wrong prefix":     prefixed("v0:", testCollection+":w001"),
		"another list's":   EncodeCursor("gadget", "g001"),
		"empty record key": EncodeCursor(testCollection, ""),
		"oversized key":    EncodeCursor(testCollection, strings.Repeat("k", MaxSortKeyBytes+1)),
	}
	for name, cursor := range cases {
		t.Run(name, func(t *testing.T) {
			rec := get(t, widgets(3), "/?cursor="+url.QueryEscape(cursor))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body.String())
			}
		})
	}
}

// A storage fault is a 500 with an opaque body, never a quietly short page,
// and the cause is in the log with the correlation id.
func TestStorageFaultIsAnOpaque500(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeList(w, r, quietLogger(), func(PageQuery) (Page[string], error) {
			return Page[string]{}, errors.New("table melted: secret-detail")
		})
	})
	rec := get(t, h, "/")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret-detail") {
		t.Fatalf("the cause leaked into the response: %s", rec.Body.String())
	}
}
