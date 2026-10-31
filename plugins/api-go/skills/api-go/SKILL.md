---
name: api-go
description: Use when adding to or reviewing the Go API of an app composed with the api-go plugin (api-runtime@1) - a new endpoint, a list endpoint, a handler's errors and logging, the OpenAPI contract, or running the API locally. The server skeleton under api/ is already in the repository as a starting point; the build adds handlers in api/internal/app and edits the shell only when it must.
metadata:
  capability: api-runtime
---

# Go API (api-go)

The API is `api/` in the repository: module `webapp`, stdlib only, `go build -o dist/api .` from
`api/`. The plugin seeds the skeleton; the app owns all of it.

## Starting points

`main.go`, `internal/shell/`, `internal/apikit/` and `internal/localauth/` are seeded once from the
plugin and are yours from then on. Prefer adding handlers in `internal/app/` to editing them, and
change them when the app needs it. `internal/shell/router.go` was written once from the
`api.middleware` and `api.routes` slots; a plugin added later does not rewire it, so add its
middleware or routes by hand.

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
- **Never emit a nil slice for an array field.** A nil slice marshals to `null`, and a client
  that maps over the field crashes on the first empty result (a real build shipped `defects: null`
  against an `openapi.yaml` array). Build with `make([]T, 0, n)`, clone with `make` and `copy`, and
  never `append([]T(nil), ...)` or `var out []T` returned as is. The same goes for a slice inside a
  nested struct or a map value. Assert the field is `[]` in the handler's test.
- **Errors are `{"error": "..."}`** (`apikit.WriteError`). A storage fault is an opaque 500 through
  `apikit.WriteFailure`; the cause is logged with the correlation id and never sent to the client. A handler that panics is contained the same way: the shell logs it with
  the correlation id and answers the JSON 500, so do not add your own recover.
- **Log with the correlation id.** `apikit.CorrelationIDFromContext(r.Context())` on every line you
  add. The shell already logs one line per request (method, path without the query string, status,
  duration); do not log request bodies or tokens.
- **Take the owner of a record from the verified identity**, never from the body or the query.
- **Reach data only through the store interface** the data-store plugin provides, so the handler
  works in the local run as well as deployed.

## Read secrets from the app's secret store

Credentials the app needs (a supplier API key, say) are SSM SecureString parameters Albitor writes
at `<prefix><NAME>`. The deploy target hands the function the prefix in the environment variable
`ALBITOR_SECRETS_PARAMETER_PREFIX` (for example `/albitor/apps/<segment>/secrets/`, ending in `/`)
and grants the function's role `ssm:GetParameter` and `ssm:GetParameters` on exactly that prefix.

- Read `os.Getenv("ALBITOR_SECRETS_PARAMETER_PREFIX")` and ask SSM (`GetParameter`, `WithDecryption`
  true) for `prefix + NAME`. Never hard-code the path, the app name or the segment: the segment is
  not the repository name, and the grant follows the variable.
- An empty variable (the local run, or a deploy with no secret store) means no secrets: treat a
  missing secret as "not configured", never as an error that takes the endpoint down.
- Never write a parameter, list a path or add an IAM policy for this: the grant is the deploy
  target's, and a secret is added through Albitor, not the repository.
- Read at request time or cache briefly; never log a value or put it in a response.

## Contributing to the slots (plugin authors)

A fragment in `api.middleware` or `api.routes` is an identifier that resolves in package
`internal/shell`: a `func(http.Handler) http.Handler` or a `func(*http.ServeMux)`. The plugin that
contributes it ships a file in `api/internal/shell/` that declares it (importing its own
package there if it needs one). The first `api.middleware` entry is the outermost, inside request
logging; a middleware that refuses unauthenticated requests must leave `GET /api/health` open.

## Run it locally

`ALBITOR_LOCAL=1 PORT=8080 go run .` from `api/`. Add `ALBITOR_LOCAL_AUTH=1` for the in-process
test issuer (`alice`, `bob`, `admin`, and one `role-<group>` identity per seeded role, holding only that group), which only issues tokens and, while it runs, binds `127.0.0.1` only. SIGTERM drains in-flight
requests (20 seconds at most) before the process exits. Verify with
`go build ./... && go vet ./... && go test ./... -count=1`.
