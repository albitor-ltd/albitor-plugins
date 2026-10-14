package store

import (
	"context"
	"sort"
	"strings"
	"sync"

	"webapp/internal/apikit"
)

// Memory is the in-process adapter for a local run and for tests: the same
// table modelled in a map. It honours every rule of the DynamoDB adapter,
// including key order (bytewise, as DynamoDB sorts strings), revisions and
// cursors, so a spec that passes locally passes deployed.
type Memory struct {
	mu   sync.Mutex
	rows map[string]map[string]Record // pk -> sk -> record
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{rows: map[string]map[string]Record{}}
}

var _ Store = (*Memory)(nil)

// Get implements Store.
func (m *Memory) Get(ctx context.Context, pk, sk string) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	if err := checkKey(pk, sk); err != nil {
		return Record{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rows[pk][sk]
	if !ok {
		return Record{}, ErrNotFound
	}
	return clone(r), nil
}

// Create implements Store.
func (m *Memory) Create(ctx context.Context, r Record) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	if err := checkWrite(r); err != nil {
		return Record{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rows[r.PK][r.SK]; ok {
		return Record{}, ErrExists
	}
	return m.put(r, 1), nil
}

// Replace implements Store.
func (m *Memory) Replace(ctx context.Context, r Record, expectedRev int64) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	if err := checkWrite(r); err != nil {
		return Record{}, err
	}
	if err := checkRev(expectedRev, false); err != nil {
		return Record{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.rows[r.PK][r.SK]
	switch {
	case !ok:
		return Record{}, ErrNotFound
	case cur.Rev != expectedRev:
		return Record{}, ErrConflict
	}
	return m.put(r, expectedRev+1), nil
}

// put stores a copy at rev and returns the stored record. The lock is held.
func (m *Memory) put(r Record, rev int64) Record {
	r.Rev = rev
	if m.rows[r.PK] == nil {
		m.rows[r.PK] = map[string]Record{}
	}
	m.rows[r.PK][r.SK] = clone(r)
	return clone(r)
}

// Delete implements Store.
func (m *Memory) Delete(ctx context.Context, pk, sk string, expectedRev int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := checkKey(pk, sk); err != nil {
		return err
	}
	if err := checkRev(expectedRev, true); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.rows[pk][sk]
	if expectedRev > 0 {
		switch {
		case !ok:
			return ErrNotFound
		case cur.Rev != expectedRev:
			return ErrConflict
		}
	}
	if ok {
		delete(m.rows[pk], sk)
		if len(m.rows[pk]) == 0 {
			delete(m.rows, pk)
		}
	}
	return nil
}

// Query implements Store.
func (m *Memory) Query(ctx context.Context, q Query) (apikit.Page[Record], error) {
	empty := apikit.Page[Record]{Items: []Record{}}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if err := checkQuery(q); err != nil {
		return empty, err
	}
	after, err := decodeCursor(q)
	if err != nil {
		return empty, err
	}

	m.mu.Lock()
	var keys []string
	for sk := range m.rows[q.PK] {
		if strings.HasPrefix(sk, q.SKPrefix) {
			keys = append(keys, sk)
		}
	}
	sort.Strings(keys)
	if q.Reverse {
		for i, j := 0, len(keys)-1; i < j; i, j = i+1, j-1 {
			keys[i], keys[j] = keys[j], keys[i]
		}
	}
	var rows []Record
	for _, sk := range keys {
		if after != "" && ((!q.Reverse && sk <= after) || (q.Reverse && sk >= after)) {
			continue
		}
		rows = append(rows, clone(m.rows[q.PK][sk]))
		if len(rows) > q.Limit {
			break
		}
	}
	m.mu.Unlock()

	return pageOf(q, rows), nil
}

// pageOf cuts rows (fetched up to Limit+1) into a page: a cursor only when a
// record follows the page.
func pageOf(q Query, rows []Record) apikit.Page[Record] {
	page := apikit.Page[Record]{Items: []Record{}}
	if len(rows) > q.Limit {
		rows = rows[:q.Limit]
		page.NextCursor = encodeCursor(q.PK, rows[len(rows)-1].SK)
	}
	page.Items = append(page.Items, rows...)
	return page
}
