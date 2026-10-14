---
name: api-go
description: Use when adding to or reviewing the Go API of an app composed with the api-go plugin (api-runtime@1) - a new endpoint, a list endpoint, a handler's errors and logging, the OpenAPI contract, or running the API locally. The server skeleton under api/ is already in the repository and managed; the build adds handlers in api/internal/app and never edits the shell.
metadata:
  capability: api-runtime
---

# Go API (api-go)

The API is `api/` in the repository: module `webapp`, stdlib only, `go build -o dist/api .` from
`api/`. The plugin owns the skeleton; the build owns `api/internal/app/`.

## What is managed (never edit)

`main.go`, `internal/shell/` (except what is generated below), `internal/apikit/` and
`internal/localauth/` are rendered from the plugin and pinned. A finding in one is the plugin's, not
yours: do not "fix" it in place. `internal/shell/router.go` is generated from the
`api.middleware` and `api.routes` slots and rewritten on every compose.

## Add an endpoint

1. A handler in its own file in `api/internal/app/`.
2. One line in `Routes` in `api/internal/app/routes.go`:
   `r.HandleFunc("GET /api/widgets", listWidgets)` (Go 1.22 method-and-path patterns).
3. The endpoint in `api/openapi.yaml`, in the same change. The web client is generated from it.

There is no `init()` registration. A handler missing from `Routes` is not served, and a pattern
registered twice panics at start.

## Rules every handler follows

- **Lists go through `apikit.ServeList`.** It enforces `limit` (default 20, max 100, out of range
  is a 400 and never clamped) and the opaque `cursor`, and answers `apikit.Page`
  (`{"items": [], "next_cursor": "..."}`) with `[]` for an empty page, never `null`. A bare array
  is a breaking change for every client the day you need a second page. Mint cursors with
  `apikit.EncodeCursor(collection, lastKey)` and read them with `apikit.DecodeCursor`; return its
  `ErrInvalidCursor` unchanged so the client gets a 400. The collection name must not contain `:`
  (both functions panic on one: it is a programming error).
- **Errors are `{"error": "..."}`** (`apikit.WriteError`). A storage fault is an opaque 500 through
  `apikit.WriteFailure`; the cause is logged with the correlation id and never sent to the client. A handler that panics is contained the same way: the shell logs it with
  the correlation id and answers the JSON 500, so do not add your own recover.
- **Log with the correlation id.** `apikit.CorrelationIDFromContext(r.Context())` on every line you
  add. The shell already logs one line per request (method, path without the query string, status,
  duration); do not log request bodies or tokens.
- **Take the owner of a record from the verified identity**, never from the body or the query.
- **Reach data only through the store interface** the data-store plugin provides, so the handler
  works in the local run as well as deployed.

## Contributing to the slots (plugin authors)

A fragment in `api.middleware` or `api.routes` is an identifier that resolves in package
`internal/shell`: a `func(http.Handler) http.Handler` or a `func(*http.ServeMux)`. The plugin that
contributes it ships a managed file in `api/internal/shell/` that declares it (importing its own
package there if it needs one). The first `api.middleware` entry is the outermost, inside request
logging; a middleware that refuses unauthenticated requests must leave `GET /api/health` open.

## Run it locally

`ALBITOR_LOCAL=1 PORT=8080 go run .` from `api/`. Add `ALBITOR_LOCAL_AUTH=1` for the in-process
test issuer (`alice`, `bob`, `admin`), which only issues tokens and, while it runs, binds `127.0.0.1` only. SIGTERM drains in-flight
requests (20 seconds at most) before the process exits. Verify with
`go build ./... && go vet ./... && go test ./... -count=1`.
