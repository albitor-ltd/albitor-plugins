package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"webapp/internal/apikit"
	"webapp/internal/identity"
)

// OwnerPrefix is the first part of every partition key an Owned handle builds.
const OwnerPrefix = "OWNER"

// ErrNoOwner means a request carries no usable owner, so no owner-scoped handle can be made.
var ErrNoOwner = errors.New("request has no owner")

// Handle is one person's records of any kind, always scoped to that person. It is optional: a
// handler may still build its own keys with Key. What the handle adds is that the scope cannot be
// forgotten or forged:
//
//   - the owner is the verified identity (identity.Owner), never an argument;
//
//   - the partition is built here, as Key("OWNER", owner, kind), so a caller cannot name another
//     owner's partition;
//
//   - another owner's record is simply not there: Get, Update and Delete answer ErrNotFound for
//     it, List never returns it, and two people can use the same id without meeting.
//
//     h, err := store.Owned(s, r)           // ErrNoOwner: answer 401
//     rec, err := h.Get(ctx, "invoice", id) // ErrNotFound for another person's invoice
//
// A kind and an id are checked like any key part (a kind may not contain the key separator), so
// request input cannot climb out of the owner's partition.
type Handle struct {
	s     Store
	owner string
}

// Owned returns the handle for the person r acts for, or ErrNoOwner when r has none.
func Owned(s Store, r *http.Request) (*Handle, error) {
	if s == nil || r == nil {
		return nil, ErrNoOwner
	}
	return ownedBy(s, identity.Owner(r))
}

// ownedBy is the one place an owner string becomes a handle, so a change in how identity answers
// the owner touches Owned alone.
func ownedBy(s Store, owner string) (*Handle, error) {
	if owner == "" {
		return nil, ErrNoOwner
	}
	if _, err := Key(OwnerPrefix, owner); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoOwner, err)
	}
	return &Handle{s: s, owner: owner}, nil
}

// Owner is the person the handle acts for.
func (h *Handle) Owner() string { return h.owner }

func (h *Handle) partition(kind string) (string, error) {
	return Key(OwnerPrefix, h.owner, kind)
}

// Get returns the owner's record of kind and id, or ErrNotFound.
func (h *Handle) Get(ctx context.Context, kind, id string) (Record, error) {
	pk, err := h.partition(kind)
	if err != nil {
		return Record{}, err
	}
	return h.s.Get(ctx, pk, id)
}

// List answers one page of the owner's records of kind. q.PK is ignored: the partition is the
// owner's. SKPrefix, Reverse, Cursor and Limit behave as in Store.Query.
func (h *Handle) List(ctx context.Context, kind string, q Query) (apikit.Page[Record], error) {
	pk, err := h.partition(kind)
	if err != nil {
		return apikit.Page[Record]{}, err
	}
	q.PK = pk
	return h.s.Query(ctx, q)
}

// Create stores a new record at revision 1 under the owner, or ErrExists when the owner already
// has that id.
func (h *Handle) Create(ctx context.Context, kind, id string, data json.RawMessage) (Record, error) {
	pk, err := h.partition(kind)
	if err != nil {
		return Record{}, err
	}
	return h.s.Create(ctx, Record{PK: pk, SK: id, Data: data})
}

// Update overwrites the owner's record if its revision is expectedRev (1 or more), as
// Store.Replace does. It never creates: no such record is ErrNotFound.
func (h *Handle) Update(ctx context.Context, kind, id string, data json.RawMessage, expectedRev int64) (Record, error) {
	pk, err := h.partition(kind)
	if err != nil {
		return Record{}, err
	}
	return h.s.Replace(ctx, Record{PK: pk, SK: id, Data: data}, expectedRev)
}

// Delete removes the owner's record, as Store.Delete does: expectedRev 0 is unconditional and
// idempotent, 1 or more is revision-checked.
func (h *Handle) Delete(ctx context.Context, kind, id string, expectedRev int64) error {
	pk, err := h.partition(kind)
	if err != nil {
		return err
	}
	return h.s.Delete(ctx, pk, id, expectedRev)
}
