---
name: data-dynamodb
description: Use when adding, changing or reviewing how the Go API of an app composed with the data-dynamodb plugin (data-store@1) stores records - a new record kind, a list endpoint over stored records, an update that must not lose a concurrent write, the table's keys, or running the API locally without AWS. The store under api/internal/store is already in the repository as a starting point; the build adds record kinds and handlers as feature packages under api/internal/ and edits the store only when it must.
metadata:
  capability: data-store
---

# DynamoDB data store (data-dynamodb)

Records live behind `store.Store` in `api/internal/store/` (package `webapp/internal/store`). It is
seeded at compose and is the app's own from then on; record kinds and handlers go in
feature packages (`api/internal/<feature>`, see the api-go skill). Deployed it is one DynamoDB table,
local it is process memory, and one conformance suite holds both to the same behaviour, so a
handler that works in a local run works deployed.

## Seeded files (the app's own)

Everything under `api/internal/store/` is seeded once: `store.go` (the interface, `Key`, errors),
`open.go` (`Open`, `TableEnv`), `memory.go` (the local adapter), `dynamodb.go` and `sigv4.go` (the
deployed adapter), their tests, and `storetest/` (the conformance suite). Extend the app around the
store rather than editing it where you can, and keep the `storetest` conformance suite passing
against both adapters. The table itself is declared in
`albitor.yaml` as `services.tables` (`data`, string `pk` and `sk`); the deploy target creates it.

## The interface

| Method | Does |
| --- | --- |
| `Create(ctx, rec)` | New record at revision 1, or `ErrExists` |
| `Get(ctx, pk, sk)` | The record, or `ErrNotFound` |
| `Replace(ctx, rec, rev)` | Overwrite if the stored revision is `rev`; `ErrConflict` if it moved, `ErrNotFound` if gone |
| `Delete(ctx, pk, sk, rev)` | `rev` 0 is unconditional and idempotent; otherwise revision-checked |
| `Query(ctx, q)` | One page of one partition, in sort-key order, with a cursor only if more follows |

A `Record` is `PK`, `SK`, `Data` (your JSON, up to 256 KiB) and `Rev` (the store's). To change a
record: `Get`, change it, `Replace` with the `Rev` you read, and on `ErrConflict` read again. Never
read-modify-write without the revision.

## Add a record kind

1. Define the Go type and `encoding/json` it into `Record.Data`. Keep that type in
   the feature's package (`api/internal/<feature>/`), or in a shared domain package when more than
   one feature reads it: a feature never imports a sibling feature.
2. For a per-user record, use the owner-scoped handle (next section). For anything else, choose the
   keys once, in one place, with `store.Key`: a partition, and an id for the sort key. Time-ordered
   ids make "newest first" a `Reverse` query.
3. Write the handler against `store.Store`. Open it once at start with
   `store.Open(os.Getenv, store.DataTable)` and hand it to your handlers.

## Per-user records: `store.Owned`

`store.Owned(s, r)` returns a handle for the person `r` acts for (the owner is
`identity.Owner(r)`, never an argument). Its operations take a `kind` and an `id`, and the
partition is built inside as `store.Key("OWNER", owner, kind)`, so the scope cannot be forgotten
or forged:

```go
h, err := store.Owned(s, r)             // store.ErrNoOwner (no identity, deployed too): answer 401
rec, err := h.Create(ctx, "note", id, data)   // stored under this person
rec, err = h.Get(ctx, "note", id)       // another person's id is ErrNotFound
page, err := h.List(ctx, "note", store.Query{Limit: q.Limit, Cursor: q.Cursor})
rec, err = h.Update(ctx, "note", id, data, rev)   // Replace with the revision you read
err = h.Delete(ctx, "note", id, rev)
```

Another person's record is not there: `Get`, `Update` and `Delete` answer `ErrNotFound`, `List`
never returns it, and the same id can be used by two people. A `kind` may not contain `#`. It is a
convenience and not a requirement: scoping by hand with `store.Key` stays valid, and the two-owner
test is the check either way.

## Add a table

The seeded `data` table (`DataTable`) holds every record kind in one partition-keyed space, which
is usually enough. For a genuinely separate keyspace, declare a second table and open it by its
logical name (lowercase, hyphens allowed, 2 to 32 characters):

```yaml
# albitor.yaml
services:
  tables:
    - name: audit
      partition_key: {name: pk, type: string}
      sort_key: {name: sk, type: string}
```

```go
audit, err := store.Open(os.Getenv, "audit") // reads ALBITOR_TABLE_AUDIT; memory when local
```

Keep the keys string `pk` and `sk`, since both adapters assume them. Hand each store to the
handlers that need it. Test with `store.NewMemory()`.

## Rules every handler follows

- **Keys come from `store.Key`.** It refuses the `#` separator and control characters, so a value
  taken from a request cannot climb into another partition. Never build a key with `+` or
  `fmt.Sprintf`.
- **The owner comes from the verified identity**, never the body or the query. Put it in the
  partition key, so one user cannot list or replay another's records: `store.Owned(s, r)` does
  this for you.
- **A per-user entity needs a factory in the two-owner test** (`ownedDataScopeRows` in
  `internal/identity/owned_data_scope_test.go`, see the api-go skill), added with the entity. Keep
  the owner in the key (`store.Owned`, or `store.Key("OWNER", owner, ...)`): two people using the same record id then
  never meet, which is what the test's same-id create checks.
- **Lists go through `apikit.ServeList` and `Store.Query`.** `Query` already speaks
  `apikit.Page`: map the records to your type and return the page. Its cursor is bound to the
  partition and prefix; a forged or foreign cursor is `store.ErrInvalidCursor` (`apikit` answers
  400). Limits are `apikit`'s: 1 to 100, never clamped.
- **Sort-key prefixes are literal.** `SKPrefix: "inv#"` matches keys that start with `inv#`; there
  are no wildcards.
- **Map errors, do not leak them.** `ErrNotFound` is a 404, `ErrExists` and `ErrConflict` are
  409, `ErrInvalid` is a 400 for caller input. Anything else is `apikit.WriteFailure`: the cause
  is logged against the correlation id and the client gets an opaque 500.
- **Do not reach past the interface.** No second DynamoDB client, no direct HTTP. A new
  capability is a new method on `Store`, implemented in both adapters and added to `storetest`.

## Local and tests

`ALBITOR_LOCAL=1` selects the in-memory store; `store.Open` refuses it on a deployed function and
beside a configured table, and refuses to start deployed without `ALBITOR_TABLE_DATA`, so it can
never fall back to memory by accident. In a Go test, use `store.NewMemory()` directly. To prove a
new `Store` implementation, run `storetest.Run(t, newStore)`.

Credentials are environment variables only (the Lambda execution role's `AWS_*` variables); the
store reads no profile or file. A write that fails with a 5xx is not retried, so treat that error
as "unknown outcome": read the record before repeating the write.

## What this store is not

One table, string keys, no secondary indexes and no multi-record transactions. If an endpoint
needs to read by something other than the partition and a sort-key prefix, or to write two records
atomically, that is a decision to make with the platform, not something to work around here.
