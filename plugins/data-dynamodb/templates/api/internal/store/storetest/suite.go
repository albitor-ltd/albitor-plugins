// Package storetest is the data-store@1 conformance suite. Run it against every
// adapter: a behaviour a handler relies on is one each adapter must have.
//
//	func TestMemoryConformance(t *testing.T) {
//		storetest.Run(t, func(t *testing.T) store.Store { return store.NewMemory() })
//	}
//
// Every case works in partitions of its own, so adapters backed by one shared
// table (DynamoDB Local) can run the suite without cleaning up between cases.
package storetest

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"webapp/internal/apikit"
	"webapp/internal/store"
)

// Run executes the suite. newStore returns a store over an empty or shared
// table; it may be called once per case.
func Run(t *testing.T, newStore func(t *testing.T) store.Store) {
	t.Helper()
	cases := []struct {
		name string
		run  func(t *testing.T, s store.Store)
	}{
		{"create then get round-trips the record", caseRoundTrip},
		{"create at a taken key is ErrExists and keeps the first record", caseCreateTwice},
		{"get of a missing record is ErrNotFound", caseGetMissing},
		{"replace needs the revision read, and bumps it", caseReplace},
		{"replace of a missing record is ErrNotFound", caseReplaceMissing},
		{"delete is idempotent unconditionally and revision-checked otherwise", caseDelete},
		{"a returned record is a copy", caseCopies},
		{"invalid keys, data and revisions are ErrInvalid", caseInvalid},
		{"query lists a partition in sort-key order, and in reverse", caseQueryOrder},
		{"query narrows by sort-key prefix, literally", caseQueryPrefix},
		{"query never crosses partitions", caseQueryIsolation},
		{"query pages without a gap or a repeat, and the last page has no cursor", casePaging},
		{"query pages in reverse", casePagingReverse},
		{"a cursor from another partition or prefix, or a forged one, is refused", caseCursors},
		{"query limits are enforced, never clamped", caseLimits},
		{"concurrent creates at one key admit exactly one", caseConcurrentCreate},
		{"concurrent replaces at one revision admit exactly one", caseConcurrentReplace},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { c.run(t, newStore(t)) })
	}
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return c
}

// partition returns a partition key no other case, run or test shares.
func partition(t *testing.T, kind string) string {
	t.Helper()
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	pk, err := store.Key("CONFORMANCE", hex.EncodeToString(b[:]), kind)
	if err != nil {
		t.Fatal(err)
	}
	return pk
}

func rec(pk, sk, data string) store.Record {
	return store.Record{PK: pk, SK: sk, Data: json.RawMessage(data)}
}

func mustCreate(t *testing.T, s store.Store, pk, sk, data string) store.Record {
	t.Helper()
	r, err := s.Create(ctx(t), rec(pk, sk, data))
	if err != nil {
		t.Fatalf("create %s/%s: %v", pk, sk, err)
	}
	return r
}

func skOf(items []store.Record) []string {
	out := make([]string, len(items))
	for i, r := range items {
		out[i] = r.SK
	}
	return out
}

func sameStrings(a, b []string) bool {
	return strings.Join(a, "\x00") == strings.Join(b, "\x00") && len(a) == len(b)
}

func caseRoundTrip(t *testing.T, s store.Store) {
	pk := partition(t, "rt")
	data := `{"title":"café \"quoted\"","n":[1,2,3],"nested":{"ok":true}}`
	created := mustCreate(t, s, pk, "a", data)
	if created.Rev != 1 || created.PK != pk || created.SK != "a" {
		t.Fatalf("created = %+v, want revision 1 at the key", created)
	}
	got, err := s.Get(ctx(t), pk, "a")
	if err != nil {
		t.Fatal(err)
	}
	var want, have any
	if err := json.Unmarshal([]byte(data), &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got.Data, &have); err != nil {
		t.Fatalf("stored data is not JSON: %v", err)
	}
	if fmt.Sprint(want) != fmt.Sprint(have) || got.Rev != 1 {
		t.Fatalf("got %s rev %d, want %s rev 1", got.Data, got.Rev, data)
	}
}

func caseCreateTwice(t *testing.T, s store.Store) {
	pk := partition(t, "twice")
	mustCreate(t, s, pk, "a", `{"v":1}`)
	if _, err := s.Create(ctx(t), rec(pk, "a", `{"v":2}`)); !errors.Is(err, store.ErrExists) {
		t.Fatalf("second create = %v, want ErrExists", err)
	}
	got, err := s.Get(ctx(t), pk, "a")
	if err != nil || string(got.Data) != `{"v":1}` || got.Rev != 1 {
		t.Fatalf("after a refused create got %s rev %d err %v, want the first record untouched", got.Data, got.Rev, err)
	}
}

func caseGetMissing(t *testing.T, s store.Store) {
	if _, err := s.Get(ctx(t), partition(t, "none"), "x"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("get = %v, want ErrNotFound", err)
	}
}

func caseReplace(t *testing.T, s store.Store) {
	pk := partition(t, "rep")
	mustCreate(t, s, pk, "a", `{"v":1}`)

	r2, err := s.Replace(ctx(t), rec(pk, "a", `{"v":2}`), 1)
	if err != nil || r2.Rev != 2 {
		t.Fatalf("replace at rev 1 = %+v, %v; want rev 2", r2, err)
	}
	// The revision the caller read is now stale.
	if _, err := s.Replace(ctx(t), rec(pk, "a", `{"v":3}`), 1); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("replace at a stale rev = %v, want ErrConflict", err)
	}
	got, _ := s.Get(ctx(t), pk, "a")
	if string(got.Data) != `{"v":2}` || got.Rev != 2 {
		t.Fatalf("a refused replace changed the record: %s rev %d", got.Data, got.Rev)
	}
	// An ahead-of-store revision is a conflict too, not a write.
	if _, err := s.Replace(ctx(t), rec(pk, "a", `{"v":9}`), 7); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("replace at a future rev = %v, want ErrConflict", err)
	}
	r3, err := s.Replace(ctx(t), rec(pk, "a", `{"v":3}`), 2)
	if err != nil || r3.Rev != 3 {
		t.Fatalf("replace at rev 2 = %+v, %v; want rev 3", r3, err)
	}
}

func caseReplaceMissing(t *testing.T, s store.Store) {
	_, err := s.Replace(ctx(t), rec(partition(t, "repmiss"), "a", `{}`), 1)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("replace of nothing = %v, want ErrNotFound (and no record created)", err)
	}
}

func caseDelete(t *testing.T, s store.Store) {
	pk := partition(t, "del")
	mustCreate(t, s, pk, "a", `{}`)

	if err := s.Delete(ctx(t), pk, "a", 5); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("delete at the wrong rev = %v, want ErrConflict", err)
	}
	if _, err := s.Get(ctx(t), pk, "a"); err != nil {
		t.Fatalf("a refused delete removed the record: %v", err)
	}
	if err := s.Delete(ctx(t), pk, "a", 1); err != nil {
		t.Fatalf("delete at rev 1 = %v", err)
	}
	if _, err := s.Get(ctx(t), pk, "a"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("get after delete = %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx(t), pk, "a", 1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("revision-checked delete of nothing = %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx(t), pk, "a", 0); err != nil {
		t.Fatalf("unconditional delete of nothing = %v, want nil", err)
	}
	mustCreate(t, s, pk, "b", `{}`)
	if err := s.Delete(ctx(t), pk, "b", 0); err != nil {
		t.Fatalf("unconditional delete = %v", err)
	}
	if _, err := s.Get(ctx(t), pk, "b"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("get after unconditional delete = %v, want ErrNotFound", err)
	}
}

func caseCopies(t *testing.T, s store.Store) {
	pk := partition(t, "copy")
	in := rec(pk, "a", `{"v":1}`)
	created, err := s.Create(ctx(t), in)
	if err != nil {
		t.Fatal(err)
	}
	in.Data[2] = 'X' // the caller's buffer must not be the store's
	created.Data[2] = 'Y'
	got, _ := s.Get(ctx(t), pk, "a")
	if !bytes.Equal(got.Data, []byte(`{"v":1}`)) {
		t.Fatalf("a caller mutated the stored record through a shared buffer: %s", got.Data)
	}
	got.Data[2] = 'Z'
	again, _ := s.Get(ctx(t), pk, "a")
	if !bytes.Equal(again.Data, []byte(`{"v":1}`)) {
		t.Fatalf("a returned record aliases the store: %s", again.Data)
	}
}

func caseInvalid(t *testing.T, s store.Store) {
	pk := partition(t, "inv")
	long := strings.Repeat("k", store.MaxKeyBytes+1)
	big := `{"x":"` + strings.Repeat("a", store.MaxDataBytes) + `"}`
	bad := []struct {
		name string
		err  error
	}{
		{"empty pk", func() error { _, e := s.Create(ctx(t), rec("", "a", `{}`)); return e }()},
		{"empty sk", func() error { _, e := s.Create(ctx(t), rec(pk, "", `{}`)); return e }()},
		{"long pk", func() error { _, e := s.Create(ctx(t), rec(long, "a", `{}`)); return e }()},
		{"long sk", func() error { _, e := s.Create(ctx(t), rec(pk, long, `{}`)); return e }()},
		{"control char in sk", func() error { _, e := s.Create(ctx(t), rec(pk, "a\x00b", `{}`)); return e }()},
		{"invalid utf-8", func() error { _, e := s.Create(ctx(t), rec(pk, "a\xffb", `{}`)); return e }()},
		{"empty data", func() error { _, e := s.Create(ctx(t), rec(pk, "a", ``)); return e }()},
		{"data with invalid utf-8", func() error { _, e := s.Create(ctx(t), rec(pk, "a", "{\"v\":\"\xff\"}")); return e }()},
		{"data not json", func() error { _, e := s.Create(ctx(t), rec(pk, "a", `{nope`)); return e }()},
		{"oversize data", func() error { _, e := s.Create(ctx(t), rec(pk, "a", big)); return e }()},
		{"replace at rev 0", func() error { _, e := s.Replace(ctx(t), rec(pk, "a", `{}`), 0); return e }()},
		{"replace at negative rev", func() error { _, e := s.Replace(ctx(t), rec(pk, "a", `{}`), -1); return e }()},
		{"delete at negative rev", s.Delete(ctx(t), pk, "a", -1)},
		{"get with empty sk", func() error { _, e := s.Get(ctx(t), pk, ""); return e }()},
		{"delete with empty pk", s.Delete(ctx(t), "", "a", 0)},
	}
	for _, b := range bad {
		if !errors.Is(b.err, store.ErrInvalid) {
			t.Errorf("%s = %v, want ErrInvalid", b.name, b.err)
		}
	}
	if _, err := s.Get(ctx(t), pk, "a"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("an invalid write left a record behind: %v", err)
	}
}

// keys inserted out of order, to prove the order is the store's.
var unordered = []string{"m", "c", "z", "a", "k", "b", "y", "d"}

func seed(t *testing.T, s store.Store, pk string, sks []string) {
	t.Helper()
	for _, sk := range sks {
		mustCreate(t, s, pk, sk, fmt.Sprintf(`{"sk":%q}`, sk))
	}
}

func caseQueryOrder(t *testing.T, s store.Store) {
	pk := partition(t, "ord")
	seed(t, s, pk, unordered)
	asc := []string{"a", "b", "c", "d", "k", "m", "y", "z"}

	page, err := s.Query(ctx(t), store.Query{PK: pk, Limit: 100})
	if err != nil || !sameStrings(skOf(page.Items), asc) || page.NextCursor != "" {
		t.Fatalf("ascending = %v cursor %q err %v, want %v and no cursor", skOf(page.Items), page.NextCursor, err, asc)
	}
	desc := []string{"z", "y", "m", "k", "d", "c", "b", "a"}
	page, err = s.Query(ctx(t), store.Query{PK: pk, Limit: 100, Reverse: true})
	if err != nil || !sameStrings(skOf(page.Items), desc) {
		t.Fatalf("descending = %v err %v, want %v", skOf(page.Items), err, desc)
	}
	empty, err := s.Query(ctx(t), store.Query{PK: partition(t, "nothing"), Limit: 5})
	if err != nil || empty.Items == nil || len(empty.Items) != 0 || empty.NextCursor != "" {
		t.Fatalf("empty partition = %+v err %v, want an empty non-nil page", empty, err)
	}
}

func caseQueryPrefix(t *testing.T, s store.Store) {
	pk := partition(t, "pre")
	// Wildcard-looking characters are literal: a prefix is not a pattern.
	seed(t, s, pk, []string{"inv#a1", "inv#a2", "inv%", "inv_x", "ord#a1", "in"})
	page, err := s.Query(ctx(t), store.Query{PK: pk, SKPrefix: "inv#", Limit: 100})
	if err != nil || !sameStrings(skOf(page.Items), []string{"inv#a1", "inv#a2"}) {
		t.Fatalf("prefix inv# = %v err %v", skOf(page.Items), err)
	}
	page, err = s.Query(ctx(t), store.Query{PK: pk, SKPrefix: "inv%", Limit: 100})
	if err != nil || !sameStrings(skOf(page.Items), []string{"inv%"}) {
		t.Fatalf("prefix inv%% = %v err %v, want only the literal match", skOf(page.Items), err)
	}
	page, err = s.Query(ctx(t), store.Query{PK: pk, SKPrefix: "zzz", Limit: 100})
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("prefix zzz = %v err %v, want none", skOf(page.Items), err)
	}
}

func caseQueryIsolation(t *testing.T, s store.Store) {
	alice, err := store.Key("OWNER", "alice", "ITEM")
	if err != nil {
		t.Fatal(err)
	}
	bob, _ := store.Key("OWNER", "bob", "ITEM")
	// Same sort keys under both owners, distinct under a shared prefix owner.
	prefix := partition(t, "iso")
	alice, bob = prefix+alice, prefix+bob
	seed(t, s, alice, []string{"1", "2"})
	seed(t, s, bob, []string{"1", "3"})
	for pk, want := range map[string][]string{alice: {"1", "2"}, bob: {"1", "3"}} {
		page, err := s.Query(ctx(t), store.Query{PK: pk, Limit: 100})
		if err != nil || !sameStrings(skOf(page.Items), want) {
			t.Fatalf("%s = %v err %v, want %v", pk, skOf(page.Items), err, want)
		}
	}
	if _, err := s.Get(ctx(t), alice, "3"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("alice reads bob's key: %v", err)
	}
	// A request-supplied owner cannot reach into another partition through Key.
	if _, err := store.Key("OWNER", "bob#ITEM", "ITEM"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("Key accepted a separator in a part: %v", err)
	}
}

func casePaging(t *testing.T, s store.Store) {
	pk := partition(t, "page")
	var all []string
	for i := 0; i < 7; i++ {
		all = append(all, fmt.Sprintf("k%02d", i))
	}
	seed(t, s, pk, all)

	for _, limit := range []int{1, 2, 3, 7, 8} {
		var got []string
		cursor := ""
		pages := 0
		for {
			page, err := s.Query(ctx(t), store.Query{PK: pk, Limit: limit, Cursor: cursor})
			if err != nil {
				t.Fatalf("limit %d: %v", limit, err)
			}
			if len(page.Items) > limit {
				t.Fatalf("limit %d: page of %d", limit, len(page.Items))
			}
			got = append(got, skOf(page.Items)...)
			pages++
			if page.NextCursor == "" {
				break
			}
			if len(page.Items) != limit {
				t.Fatalf("limit %d: a cursor on a short page of %d", limit, len(page.Items))
			}
			cursor = page.NextCursor
			if pages > 20 {
				t.Fatalf("limit %d: paging does not terminate", limit)
			}
		}
		if !sameStrings(got, all) {
			t.Fatalf("limit %d: paged %v, want %v", limit, got, all)
		}
		// A cursor means at least one more record: an exactly-full last page has none.
		if limit == 7 && pages != 1 {
			t.Fatalf("limit 7 over 7 records took %d pages, want 1 (no empty trailing page)", pages)
		}
	}
}

func casePagingReverse(t *testing.T, s store.Store) {
	pk := partition(t, "pagerev")
	seed(t, s, pk, unordered)
	var got []string
	cursor := ""
	for i := 0; i < 20; i++ {
		page, err := s.Query(ctx(t), store.Query{PK: pk, Limit: 3, Reverse: true, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, skOf(page.Items)...)
		if cursor = page.NextCursor; cursor == "" {
			break
		}
	}
	if want := []string{"z", "y", "m", "k", "d", "c", "b", "a"}; !sameStrings(got, want) {
		t.Fatalf("reverse paging = %v, want %v", got, want)
	}
}

func caseCursors(t *testing.T, s store.Store) {
	pkA, pkB := partition(t, "curA"), partition(t, "curB")
	seed(t, s, pkA, []string{"inv#1", "inv#2", "inv#3"})
	seed(t, s, pkB, []string{"inv#1", "inv#2", "inv#3"})
	page, err := s.Query(ctx(t), store.Query{PK: pkA, SKPrefix: "inv#", Limit: 1})
	if err != nil || page.NextCursor == "" {
		t.Fatalf("setup page = %+v err %v", page, err)
	}
	forged := []struct {
		name string
		q    store.Query
	}{
		{"cursor of another partition", store.Query{PK: pkB, SKPrefix: "inv#", Limit: 1, Cursor: page.NextCursor}},
		{"cursor of another prefix", store.Query{PK: pkA, SKPrefix: "other", Limit: 1, Cursor: page.NextCursor}},
		{"not base64", store.Query{PK: pkA, Limit: 1, Cursor: "!!!"}},
		{"base64 of nothing we issued", store.Query{PK: pkA, Limit: 1, Cursor: "aGVsbG8"}},
		{"oversize", store.Query{PK: pkA, Limit: 1, Cursor: strings.Repeat("A", 4096)}},
	}
	for _, f := range forged {
		_, err := s.Query(ctx(t), f.q)
		if !errors.Is(err, apikit.ErrInvalidCursor) {
			t.Errorf("%s = %v, want apikit.ErrInvalidCursor", f.name, err)
		}
	}
	// And the genuine cursor still works against its own partition and prefix.
	next, err := s.Query(ctx(t), store.Query{PK: pkA, SKPrefix: "inv#", Limit: 1, Cursor: page.NextCursor})
	if err != nil || !sameStrings(skOf(next.Items), []string{"inv#2"}) {
		t.Fatalf("resume = %v err %v, want inv#2", skOf(next.Items), err)
	}
}

func caseLimits(t *testing.T, s store.Store) {
	pk := partition(t, "lim")
	seed(t, s, pk, []string{"a", "b"})
	for _, limit := range []int{0, -1, apikit.MaxPageLimit + 1} {
		if _, err := s.Query(ctx(t), store.Query{PK: pk, Limit: limit}); !errors.Is(err, store.ErrInvalid) {
			t.Errorf("limit %d = %v, want ErrInvalid", limit, err)
		}
	}
	if _, err := s.Query(ctx(t), store.Query{PK: "", Limit: 1}); !errors.Is(err, store.ErrInvalid) {
		t.Errorf("empty pk = %v, want ErrInvalid", err)
	}
	if page, err := s.Query(ctx(t), store.Query{PK: pk, Limit: apikit.MaxPageLimit}); err != nil || len(page.Items) != 2 {
		t.Errorf("max limit = %v err %v", skOf(page.Items), err)
	}
}

// race runs fn n times at once and returns the errors.
func race(n int, fn func(i int) error) []error {
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = fn(i)
		}()
	}
	close(start)
	wg.Wait()
	return errs
}

func caseConcurrentCreate(t *testing.T, s store.Store) {
	pk := partition(t, "racec")
	errs := race(8, func(i int) error {
		_, err := s.Create(ctx(t), rec(pk, "a", fmt.Sprintf(`{"writer":%d}`, i)))
		return err
	})
	won := 0
	for _, err := range errs {
		switch {
		case err == nil:
			won++
		case !errors.Is(err, store.ErrExists):
			t.Errorf("a losing create = %v, want ErrExists", err)
		}
	}
	if won != 1 {
		t.Fatalf("%d creates won, want exactly 1", won)
	}
}

func caseConcurrentReplace(t *testing.T, s store.Store) {
	pk := partition(t, "racer")
	mustCreate(t, s, pk, "a", `{"v":0}`)
	errs := race(8, func(i int) error {
		_, err := s.Replace(ctx(t), rec(pk, "a", fmt.Sprintf(`{"v":%d}`, i+1)), 1)
		return err
	})
	won := 0
	for _, err := range errs {
		switch {
		case err == nil:
			won++
		case !errors.Is(err, store.ErrConflict):
			t.Errorf("a losing replace = %v, want ErrConflict", err)
		}
	}
	if won != 1 {
		t.Fatalf("%d replaces won at revision 1, want exactly 1", won)
	}
	if got, _ := s.Get(ctx(t), pk, "a"); got.Rev != 2 {
		t.Fatalf("revision after the race = %d, want 2", got.Rev)
	}
}
