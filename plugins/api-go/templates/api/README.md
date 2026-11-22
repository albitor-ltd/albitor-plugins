# api

The Go API. It runs behind Lambda Web Adapter in the
delivered app and as a plain HTTP server locally:

```sh
ALBITOR_LOCAL=1 PORT=8080 go run .
```

## Layout

| Path | Whose | What |
|---|---|---|
| `main.go`, `internal/shell/`, `internal/apikit/`, `internal/localauth/` | Yours | The server skeleton |
| `internal/shell/router.go` | Yours | Lists the middleware chain and any extra route groups |
| `internal/outbound/` | Yours | The guarded fetch for any URL a request, record or config supplied. Never fetch such a URL with a bare `http.Client` |
| `internal/apptest/` | Yours | The shared HTTP test helper feature packages use. Features do not edit it |
| `internal/app/registry.go` | Yours | The wiring point: an import line and a `registrars` line per feature, append-only, no code |
| `internal/app/routes.go` | Yours | `Routes`, which walks `registrars`. Not edited per feature |
| `internal/<feature>/` | Yours | One feature, its own Go package: its handlers, an exported `Register`, its tests |
| `openapi.yaml` | Yours | The API contract. Add each endpoint in the same change as its handler |

## Adding an endpoint

Give each feature its own package, `internal/widgets`: handlers in `widgets.go`, an exported
`Register(r *http.ServeMux)` in `routes.go`, tests in `widgets_test.go` (using
`apptest.Serve(t, Register, ...)`). Append `import "webapp/internal/widgets"` and `widgets.Register,`
to `internal/app/registry.go`, and describe the endpoints in `openapi.yaml`:

```go
func Register(r *http.ServeMux) {
	r.HandleFunc("GET /api/widgets", listWidgets)
}
```

There is no `init()` registration and no generator: a feature not listed in `registrars` is not
served. Features edit no other shared file, and two features never share a package directory, so
parallel features rebase cleanly: two features added at once merge as a text union in
`registry.go` (tested), which is why it holds imports and the list and no code, with its comment
lines kept as padding between the two. A feature imports shared packages only,
never `app` and never a sibling feature (a feature that needs another's data goes through a shared
domain or store package); `TestFeaturesDoNotImportEachOther` checks it.
Handlers reach data only through the app's store interface, never a
client of their own, so they work in the local run as well as deployed.

## Fetching a URL from a request

Fetch any URL a person or a config supplied through `internal/outbound` (`outbound.New(...).Get`),
never a bare `http.Client`: it refuses non-http(s) schemes and loopback, private, link-local
(cloud metadata) and other reserved addresses, on the resolved IP and on every redirect.

## Schedules and queues

A `schedules` or `queues` entry in `albitor.yaml` runs this same binary. The platform delivers each
fire as `POST /events`: `{"schedule": "<name>"}` for a schedule, SQS's `{"Records": [...]}` for a
queue. Register that route from its own feature package's `Register`, dispatch on the body, run the job before answering, and
answer 5xx if it failed: a 5xx fails the invocation and Lambda retries it, while any other status
counts as success. An unknown name is a 400. The route is not reachable from the public API, but
act only on a declared name anyway, and make every job idempotent: a schedule can fire twice and a
queue delivers at least once. The platform owns the schedule tick (locally too), so never add a
ticker of your own, and do not put work a user is waiting on on a schedule.

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

A role of `services.auth.groups` is seeded the same way, `POST /api/_local/auth/seed {"role":"manager"}`
(idempotent), and then signs in as `role-manager`: an identity holding that group and no other. The
`auth@2` interface (`roles`, `seed`, `signInAs`) drives this in a spec.

It only issues tokens (keys at `/api/_local/auth/.well-known/jwks.json`); verifying them is the auth
middleware's job. It refuses to start without `ALBITOR_LOCAL=1`, on a deployed function, or
beside a configured user pool.

## Verify

```sh
go build ./... && go vet ./... && go test ./... -count=1
```
