# api

The Go API, from the `api-go` plugin (`api-runtime@1`). It runs behind Lambda Web Adapter in the
delivered app and as a plain HTTP server locally:

```sh
ALBITOR_LOCAL=1 PORT=8080 go run .
```

## Layout

| Path | Whose | What |
|---|---|---|
| `main.go`, `internal/shell/`, `internal/apikit/`, `internal/localauth/` | Managed | The server skeleton. Rendered by the plugin; do not edit |
| `internal/shell/router.go` | Generated | Places the `api.middleware` and `api.routes` slots. Rewritten on compose |
| `internal/app/routes.go` | Yours | Registers your handlers. Add a file per handler and one line here |
| `openapi.yaml` | Yours | The API contract. Add each endpoint in the same change as its handler |

## Adding an endpoint

Write the handler in `internal/app/`, register it in `Routes`, describe it in `openapi.yaml`:

```go
func Routes(r *http.ServeMux) {
	r.HandleFunc("GET /api/widgets", listWidgets)
}
```

There is no `init()` registration and no generator: a handler not listed in `Routes` is not served.
Handlers reach data only through the store interface the data-store plugin gives the app, never a
client of their own, so they work in the local run as well as deployed.

## Patterns to keep

- **Every list is paginated.** A list endpoint takes `limit` (default 20, maximum 100, out of range
  is a 400) and an opaque `cursor`, and answers `{"items": [], "next_cursor": "..."}`, never a bare
  array. Use `apikit.ServeList`; it is the only way to ship a list without a limit and a cursor.
  `next_cursor` is absent on the last page.
- **Errors are `{"error": "..."}`**, written with `apikit.WriteError`. A storage fault is an opaque
  500 (`apikit.WriteFailure`): the cause goes in the log with the correlation id, not the response.
- **Every request is logged once**, as a JSON line with the correlation id, method, path (no query
  string), status and duration; a 5xx logs at `error`. Put the id on your own log lines with
  `apikit.CorrelationIDFromContext`.

## Local run

`ALBITOR_LOCAL=1` boots with no AWS and no configuration. Adding `ALBITOR_LOCAL_AUTH=1` starts an
in-process stand-in for the user pool, so specs sign in as `alice`, `bob` and `admin`:

```sh
ALBITOR_LOCAL=1 ALBITOR_LOCAL_AUTH=1 PORT=8080 go run .
curl -s -X POST localhost:8080/api/_local/auth/token -d '{"identity":"alice"}'
```

It only issues tokens (keys at `/api/_local/auth/.well-known/jwks.json`); verifying them is the auth
plugin's middleware. It refuses to start without `ALBITOR_LOCAL=1`, on a deployed function, or
beside a configured user pool.

## Verify

```sh
go build ./... && go vet ./... && go test ./... -count=1
```
