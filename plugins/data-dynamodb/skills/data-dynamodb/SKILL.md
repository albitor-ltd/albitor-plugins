---
name: data-dynamodb
description: Use when adding, changing or reviewing how the Go API of an app composed with the data-dynamodb plugin (data-store@1) stores records - a new record kind, a list endpoint over stored records, an update that must not lose a concurrent write, the table's keys, or running the API locally without AWS. The store under api/internal/store is already in the repository and managed; the build adds record kinds and handlers in api/internal/app and never edits the store.
metadata:
  capability: data-store
---

# DynamoDB data store (data-dynamodb)

Records live behind `store.Store` in `api/internal/store/` (package `webapp/internal/store`). The
plugin owns that directory; the build owns `api/internal/app/`. Deployed it is one DynamoDB table,
local it is process memory, and one conformance suite holds both to the same behaviour, so a
handler that works in a local run works deployed.

## What is managed (never edit)

Everything under `api/internal/store/`, including `storetest/`. A finding there is the plugin's,
not yours: do not "fix" it in place. The table itself is declared in `albitor.yaml` as
`services.tables` (`data`, string `pk` and `sk`); the deploy target creates it.

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
   `api/internal/app/`.
2. Choose the keys once, in one place, with `store.Key`:
   `store.Key("OWNER", owner, "ITEM")` for the partition, and an id for the sort key. Time-ordered
   ids make "newest first" a `Reverse` query.
3. Write the handler against `store.Store`. Open it once at start with
   `store.Open(os.Getenv, store.DataTable)` and hand it to your handlers.

## Rules every handler follows

- **Keys come from `store.Key`.** It refuses the `#` separator and control characters, so a value
  taken from a request cannot climb into another partition. Never build a key with `+` or
  `fmt.Sprintf`.
- **The owner comes from the verified identity**, never the body or the query. Put it in the
  partition key, so one user cannot list or replay another's records.
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

## What this store is not

One table, string keys, no secondary indexes and no multi-record transactions. If an endpoint
needs to read by something other than the partition and a sort-key prefix, or to write two records
atomically, that is a decision to make with the platform, not something to work around here.
