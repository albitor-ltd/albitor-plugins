---
name: api-go
description: Use when adding to or reviewing the Go API of an app composed with the api-go plugin (api-runtime@1) - a new endpoint, a list endpoint, a handler's errors and logging, the OpenAPI contract, or running the API locally. The server skeleton under api/ is already in the repository as a starting point; the build adds handlers in api/internal/app and edits the shell only when it must.
metadata:
  capability: api-runtime
---

# Go API (api-go)

The API is `api/` in the repository: module `webapp`, stdlib only, `go build -o dist/api .` from
`api/`. The skeleton is seeded at compose; the app owns all of it from then on.

## Seeded files (the app's own)

These are written once and are plain app code. Prefer adding handlers in `internal/app/` to editing
them, and change them when the app needs it.

- `main.go`, `go.mod`, `README.md`
- `internal/shell/` (`boot.go`, `shell.go`, `shell_test.go`, `router.go`): server, health route,
  JSON 404, request logging, middleware chain
- `internal/apikit/`: JSON and error writers, correlation ids, paging
- `internal/localauth/`: the local test issuer
- `internal/app/routes.go`: where your handlers are registered
- `openapi.yaml`: the base contract (an auth plugin's skill carries the variant with `GET /api/me`)

`internal/shell/router.go` was written once from the middleware and route groups composed at that
time; one added later does not rewire it, so add its middleware or routes by hand.

## Add an endpoint

1. A handler in its own file in `api/internal/app/`.
2. One line in `Routes` in `api/internal/app/routes.go`:
   `r.HandleFunc("GET /api/widgets", listWidgets)` (Go 1.22 method-and-path patterns).
3. The endpoint in `api/openapi.yaml`, in the same change. The web client is generated from it.

A worked example, `internal/app/widgets.go`. The handlers take no dependencies, so they register
directly in `Routes(r *http.ServeMux)`; an app that needs a store opens it once and keeps it in a
package variable or a struct that `Routes` builds:

```go
package app

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"webapp/internal/apikit"
)

type Widget struct {
	Name string `json:"name"`
}

func listWidgets(w http.ResponseWriter, r *http.Request) {
	apikit.ServeList(w, r, slog.Default(), func(q apikit.PageQuery) (apikit.Page[Widget], error) {
		// Read one page using q.Cursor and q.Limit; mint next_cursor with apikit.EncodeCursor.
		return apikit.Page[Widget]{Items: make([]Widget, 0)}, nil
	})
}

func createWidget(w http.ResponseWriter, r *http.Request) {
	var in Widget
	body := http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(body).Decode(&in); err != nil || in.Name == "" {
		apikit.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	// Take the owner from the verified identity, never the body; write through the store.
	apikit.WriteJSON(w, http.StatusCreated, in)
}
```

Register them in `Routes` (`GET /api/widgets`, `POST /api/widgets`), describe both in
`openapi.yaml`, and test them with `httptest` against `shell.Handler`.

There is no `init()` registration. A handler missing from `Routes` is not served, and a pattern
registered twice panics at start.

## `POST /events`: schedules and queues

An `albitor.yaml` `schedules` or `queues` entry runs this same binary (`handler: api`). The platform
runs it behind Lambda Web Adapter in pass-through mode: the trigger is not HTTP, so the adapter
delivers it to the one server as `POST /events`. An app that declares either must serve that route,
or every fire (and every queue message, which then lands in the dead-letter queue) hits a 404. The
body says which trigger it is:

- a schedule: `{"schedule": "<name>"}`, `<name>` the entry's `name`;
- a queue: `{"Records": [...]}`, SQS's own event (a batch of 1 to 10 messages, or an empty
  `Records` list from the deploy probe).

```go
r.HandleFunc("POST /events", handleEvents)

func handleEvents(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil { /* 400 */ }
	if ev, err := gateway.ParseEvent(body); err == nil { // a queue event
		var failed []string
		for _, m := range ev.Messages {
			if err := handleMessage(r.Context(), m); err != nil {
				failed = append(failed, m.ID) // log it with the correlation id
			}
		}
		gateway.WriteResult(w, failed) // 200, listing only the failures
		return
	}
	var s struct{ Schedule string `json:"schedule"` }
	_ = json.Unmarshal(body, &s)
	switch s.Schedule { // answer 400 for a name you do not know, never a silent 200
	}
}
```

- **Schedules.** Answer 2xx when the job ran and 5xx when it failed. The platform makes a 5xx fail
  the invocation (`AWS_LWA_ERROR_STATUS_CODES=500-599`), so Lambda retries the run and the failure
  is reported; any other status, a 4xx included, counts as success and is not retried.
- **The job runs inside the invocation.** Lambda freezes the environment once the response is
  written and the function timeout is 30 seconds. Do the work before answering.
- **Idempotent, always.** A schedule can fire twice (a retry after a 5xx) and a standard queue
  delivers at least once and unordered, so a second run of the same job or message must change
  nothing: key writes on a record id or a conditional write, check "already done" first, and never
  assume order between messages.
- **Untrusted input.** `/events` answers 404 at the public API, so only the platform (the
  scheduler, the queue's event source mapping, the deploy probe) reaches it. Still act only on a
  schedule or queue name the app declares, and carry a reference (an object key, a record id) in a
  payload, not data, re-checking ownership of what it names.
- Keep `/events` out of `openapi.yaml`: it is not part of the client contract.

### No app-side schedule loop

The platform owns the tick. Under `ALBITOR_LOCAL=1` it POSTs each declared schedule to the local
server's `/events` every second so a handler is exercised; deployed, the scheduler fires it at its
rate (15 minutes at the fastest). Never add a `time.Ticker`, a goroutine loop or a start-up
trigger that calls the job: it runs twice locally, runs on every cold start and on every instance.
Do not plan **work a user is waiting on** on a schedule: a schedule fires at most every 15 minutes
whether or not there is work, so the user waits up to that long. That work belongs inline (within
the 30 second request) or on a queue.

### Queues

A `queues` entry is an SQS standard queue with one consumer, this binary at `POST /events`. Declare
it only when the app's deploy target renders `queues`. First copy the
`deploy-hosted` skill's gateway example to `api/internal/gateway/` (`cp -r`, then edit; nothing
seeds it). Use its queue half (`queue.go`; it does not go through the gateway):

```go
q, err := gateway.OpenQueues(os.Getenv)   // in main; a local run gives the in-process queue
q.Register(handler)                          // local only, once the http.Handler exists
err = q.Send(ctx, "transcribe", jobRef{ID: id}, gateway.SendOptions{})
```

- **Send** from a route, a schedule or another consumer with `Send` or `SendBatch` (up to 10). The
  platform sets `ALBITOR_QUEUE_<NAME>_URL` (upper-cased, `-` as `_`) from the declared name; an
  undeclared queue is `ErrQueueNotConfigured`. A message is at most 256 KiB, so queue a reference.
  `SendOptions.Delay` (0 to 900 s) holds delivery back.
- **Re-enqueue is bounded.** The envelope carries `attempt` and `deadline`. A handler that
  re-enqueues passes the message it is handling as `SendOptions{Parent: &m, Delay: ...}`: `attempt`
  increments, the deadline stays, and `Send` returns `ErrAttemptsExceeded` past 50 or
  `ErrDeadlinePassed` after it. The first send sets the deadline from `Horizon` (at most one hour,
  which is also the default). On either error stop and mark the job failed; do not retry around it.
- **Failure semantics.** `ParseEvent` returns the messages (`ID`, `Queue`, `Payload`, `Attempt`,
  `Deadline`, `ReceiveCount`); dispatch on `m.Queue`. Collect the ids of messages that failed and
  answer with `gateway.WriteResult(w, failed)`: a 2xx listing them in `batchItemFailures` retries
  only those, a 2xx listing none deletes the batch, and a 5xx or a timeout returns the whole
  batch. After `max_receives` deliveries a message goes to the dead-letter queue, which nobody
  reads, so do not rely on it. A message in `ev.Malformed` can never succeed: log it and leave it
  out of the failures. Whether the adapter returns the response body to Lambda is still to be
  confirmed on the first deployed queue; if not, `WriteResult` answers 5xx for any failure and the
  whole batch retries, so handlers must be idempotent either way. Only a 5xx counts as an error:
  a 4xx is a success and deletes the batch, so a handler that wants a retry must answer 5xx (or
  use `batchItemFailures`), never a 4xx.
- **Empty batch.** The deploy probes each consumer with `{"Records": []}`. Answer 2xx and do
  nothing; a 404 there fails the deploy.
- **Local run.** `OpenQueues` returns the in-process queue under `ALBITOR_LOCAL=1`: `Send` calls
  your handler at `POST /events` with an SQS-shaped body, one message per call, honours `Delay`,
  retries up to three times and logs a message it drops. It keeps nothing across restarts and is
  not SQS: do not test visibility timeouts or dead-lettering against it.

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

Credentials the app needs (a supplier API key, say) are SSM SecureString parameters the platform writes
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
