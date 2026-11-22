---
name: api-go
description: Use when adding to or reviewing the Go API of an app composed with the api-go plugin (api-runtime@1) - a new endpoint, a list endpoint, a handler's errors and logging, the OpenAPI contract, or running the API locally. The server skeleton under api/ is already in the repository as a starting point; the build adds each feature as its own Go package under api/internal/ and edits the shell only when it must.
metadata:
  capability: api-runtime
---

# Go API (api-go)

The API is `api/` in the repository: module `webapp`, stdlib only, `go build -o dist/api .` from
`api/`. The skeleton is seeded at compose; the app owns all of it from then on.

## Seeded files (the app's own)

These are written once and are plain app code. Prefer adding a feature package (`internal/<feature>`)
to editing them, and change them when the app needs it.

- `main.go`, `go.mod`, `README.md`
- `internal/shell/` (`boot.go`, `shell.go`, `shell_test.go`, `router.go`): server, health route,
  JSON 404, request logging, middleware chain
- `internal/apikit/`: JSON and error writers, correlation ids, paging
- `internal/localauth/`: the local test issuer
- `internal/ownedscope/`: the two-owner contract test's helper (see "The two-owner test"); an auth plugin
  seeds the test that uses it, `internal/identity/owned_data_scope_test.go`
- `internal/outbound/`: the guarded fetch for any URL the app did not write (see "Fetching a URL a
  person or a config supplied")
- `internal/app/registry.go`: the one wiring point. Per feature it gets an import line and a line
  in the `registrars` list, both append-only. It holds nothing else (no code, no `func`), which is
  what lets a text union merge two features' lines
- `internal/app/routes.go`: `Routes`, which only walks `registrars`; not edited per feature
- `internal/app/routes_test.go`, `layout_test.go`: a mount check, and the check that no feature
  imports app or a sibling feature; features do not edit them
- `internal/apptest/`: the shared HTTP test helper (`Serve`, `Do`) feature packages use; features
  do not edit it
- `openapi.yaml`: the base contract (an auth plugin's skill carries the variant with `GET /api/me`)

`internal/shell/router.go` was written once from the middleware and route groups composed at that
time; one added later does not rewire it, so add its middleware or routes by hand.

## Add an endpoint

Each feature is its own Go package, `internal/<feature>`, so two build nodes writing different
features never write the same package directory, and their rebases never touch the same lines. For
a feature `widgets`, in the worked example `examples/api/internal/widgets/` (copy it, rename it):

1. Handlers in `api/internal/widgets/widgets.go` (`package widgets`).
2. Registration in `api/internal/widgets/routes.go`: an exported `Register(r *http.ServeMux)` that
   calls `r.HandleFunc("GET /api/widgets", listWidgets)` (Go 1.22 method-and-path patterns).
3. Tests in `api/internal/widgets/widgets_test.go`, using `apptest.Serve(t, Register, method, path,
   body)` from `webapp/internal/apptest`.
4. Two lines appended in `api/internal/app/registry.go`, the only shared lines a feature adds: an
   `import "webapp/internal/widgets"` line under the "one import line per feature" comment, and
   `widgets.Register,` in `registrars`. Write the import as its own one-line statement (no block,
   no alias). Append; never reorder, reflow or edit another feature's line, and never add a
   feature's route to `Routes` itself. Keep `registry.go` to the imports and the list: adding a
   function or any other code to it stops the resolver unioning it.
5. The endpoint in `api/openapi.yaml`, in the same change. The web client is generated from it.

```go
// api/internal/app/registry.go after adding widgets
// one import line per feature, appended below
import "webapp/internal/widgets"

// ... the comment lines of the seeded file stay here, at least four of them, so the import and the
// list below are never within three lines of each other (see the note under this block)

var registrars = []registrar{
	// one line per feature, appended below
	widgets.Register,
}
```

Keep at least four unchanged lines (the seeded comment lines) between the import lines and
`var registrars`: a three-way merge treats changes closer than that as one hunk, and two features
added at once would then tangle an import into the list. Never delete that comment block.

A feature node declares its own `internal/<feature>/` files and the one-line `registry.go` addition
(an import line and a `Register` line). Do not rely on the platform exempting the func-free registry
from its same-directory ordering: write only what the feature needs.

Do not put a second feature's handlers, routes or tests in a feature's package, and do not edit
`internal/apptest`: a helper only one feature needs goes in that feature's own `_test.go` file.
Albitor's mechanical rebase resolver takes a `registry.go` whose base has no `func` and whose
change only adds import or `name,` lines, and merges two concurrent additions as a text union (the
verify script tests that the result parses), so they land without an agent; `routes.go` is not unioned and is left alone.
Albitor still orders any two nodes that write the same package directory, so a package per feature
is what lets features be built in parallel.

### One feature, one package, no sideways imports

- **A feature imports only shared packages**: `apikit`, `identity`, `store`, `outbound`, a shared
  domain package. Never `app` (it imports every feature, so that is an import cycle), and never a
  sibling feature (`internal/orders` importing `internal/widgets` fails to compile the day
  `widgets` imports `orders`, and ties two nodes' work together before then).
- **A feature that needs another feature's data goes through a shared package.** Put the type and
  its read or write functions in a package neither feature owns (`internal/catalogue`, or the
  `store` with a typed accessor), and have both import it. Do the same for a helper two features
  share. Do not copy the sibling's code to avoid the import.
- **Shared test helpers live in `internal/apptest`**, which imports nothing of the app, so any
  feature's tests can use it. A test that needs the whole app mounted belongs in `internal/app`
  (`apptest.Serve(t, Routes, ...)`), not in a feature.
- `TestFeaturesDoNotImportEachOther` (`internal/app/layout_test.go`) fails when a package
  `registry.go` imports imports `app` or another such package. Fix the import; do not edit the test.

A worked example, `internal/widgets/widgets.go`. The handlers take no dependencies, so `Register`
registers them directly; an app that needs a store opens it once and keeps it in a package
variable or a struct that `Register` builds:

```go
package widgets

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
	// Take the owner from the verified identity, never the body; write through the store
	// (with data-dynamodb, store.Owned(s, r) scopes every call to that owner).
	apikit.WriteJSON(w, http.StatusCreated, in)
}
```

```go
// widgets/routes.go
package widgets

import "net/http"

func Register(r *http.ServeMux) {
	r.HandleFunc("GET /api/widgets", listWidgets)
	r.HandleFunc("POST /api/widgets", createWidget)
}
```

```go
// widgets/widgets_test.go
func TestListWidgetsEmpty(t *testing.T) {
	rec := apptest.Serve(t, Register, http.MethodGet, "/api/widgets", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}
```

Describe both endpoints in `openapi.yaml`. Test them with `apptest.Serve` (it mounts one feature's
`Register` on a plain mux), or with `httptest` against `shell.Handler` from outside the package.

There is no `init()` registration. A feature missing from `registrars` is not served, and a pattern
registered twice panics at start (`TestRoutesMountWithoutConflict` catches it).

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
// internal/events/routes.go: its own feature package, wired with events.Register in registrars
func Register(r *http.ServeMux) { r.HandleFunc("POST /events", handleEvents) }

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

## Shape endpoints for the screen

On the hosted lane, the function runs at most 10 requests at once and the API takes 50 a second, so a page that
makes one request per row overflows on a normal list. Do the work in the handler, in one
invocation: see the web-app-architecture `screen-data` pattern.

- **A list returns what the list renders.** Embed the summaries and counts a row shows (owner name,
  open-task count) in each `items` entry, built with one `BatchGetItem` over the distinct related
  ids or a counter kept on the record. Never make the client fetch `GET /api/widgets/{id}` for
  every row.
- **A screen that spans entities gets its own endpoint** (`GET /api/dashboard`) that reads them
  all and answers one document, declared in `openapi.yaml` with the rest.
- **When a batch read is unavoidable, take `ids`.** `GET /api/widgets?ids=a,b,c` answers every
  id in one invocation. It is a separate handler, not `apikit.ServeList`: it caps the ids at the
  list maximum (100), answers 400 over it (never truncating), and returns the items with no
  cursor, so do not bolt `ids` onto `ServeList`. Skip ids the caller does not own and omit them
  from the answer, and keep the order of `ids`. It is a read: it takes the owner from the
  verified identity like any other.
- Parallel reads inside the handler (`errgroup`, bounded to a few) are cheap; parallel requests
  from the client are not.

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
- **Take the owner of a record from the verified identity**, never from the body or the query. With
  the data-dynamodb plugin, `store.Owned(s, r)` does it for you (optional; scoping by hand is valid).
- **Fetch a request-supplied or user-supplied URL only through `internal/outbound`**, never a bare
  `http.Client` (see "Fetching a URL a person or a config supplied").
- **Reach data only through the store interface** the data-store plugin provides, so the handler
  works in the local run as well as deployed.

## The two-owner test

`internal/identity/owned_data_scope_test.go` (`TestOwnedDataScope`) runs in process against `app.Routes`
with two signed-in people. The platform runs the same test names (`go test -run '^TestOwnedDataScope'`)
and judges the report, so this test covers the operations the platform requires (judged on the merged
report of every package, and on the contract after the plan's `shared` declarations: a plan-level
`shared` entity is not owed a row, and the platform's required set is the answer when they differ).
For every collection of `openapi.yaml` (a `POST` with no path parameter that is neither public nor
admin-only, with records under it) it covers the create, the list and every route under one record (the
record itself and actions such as `POST /things/{id}/archive`), except a collection whose create names
(`x-entity`) an entity the `x-data-model` declares `shared`. A route naming no entity is still covered:
its row's `Entity` is the collection's path. The first person makes a record, and the second person must
not be able to read, update or delete it or find it in a list, and must not be able to take it over by
creating a record with the same id. The `servers[0]` base path is applied to the paths (the report
names `/api/things/{id}` where the contract has `/things/{id}` under a `/api` server).

- **Provenance and nested routes are advisory evidence.** The helper stamps every row it writes with
  its name, the kit release, a digest of its own source (`provenance.go`, `nested.go` and the other
  helper files) and a digest of the row, and lists the nested routes it drove in the report's
  `nested` array. A nested route is one under a record that takes the record's id and another
  (`GET /api/lists/{id}/items/{itemId}`): the second person makes a record of their own with the same
  factory and calls the route with the first person's parent id and their own child id, and must be
  refused (404 or 403). Give each person ids of their own in the factory (derive them from
  `as.Identity.Name`), and put every id of the record in `Record.Params` by path-parameter name;
  a route with a body needs a `Probe`. A `not-driven` row says what is missing. The stamp is not
  tamper-proof: anyone who owns the app's files can edit the helper. It lets the platform tell a
  report the helper wrote from one typed by hand, and a helper that is the kit's from one that was
  changed. Neither the stamp nor the nested rows ever fail the test. Do not edit the files under
  `internal/ownedscope/` or the report.
- **Keep this test.** It is part of the app. It runs with `go test ./...`; set
  `OWNED_DATA_SCOPE_REPORT` to a path to also get its JSON report, one entry per route covered
  (`refused`, `leaked` or `waived` with a reason). The helper's own tests are named `TestScopeHelper...`
  and never write that file; do not name an app test `TestOwnedDataScope...` unless it is meant to be
  judged.
- **Register a factory for every collection you add**, in `ownedDataScopeRows`, in the same change as
  the collection. A factory makes one record through the app's own routes as the first person
  and returns its `ID` and a valid create body (and, where they must differ, an `Update` body and a
  `Collide` body). The test never guesses a body. A route that is not create, list, read, update
  or delete (an action on a record, `POST` or `GET`) takes a `Probe`.
- **"Refused" means** the second person got a 404 or a 403 and the first person's record was left
  exactly as it was; for a list it means the record is not in the answer; for the same-id create it
  means the first person's record is intact (a record in the second person's own keyspace is fine).
  Any other answer, a 2xx included, is "leaked" and fails the test.
- **A route with no factory row fails the test.** Add the row.
- **Admin-only routes are in the test.** For every operation the contract declares admin-only
  (`x-admin-only: true`, `x-required-groups` or `x-groups` naming `admin` in any case, on the operation
  or its path item, or a `security` requirement whose every alternative asks for an admin scope) add a
  probe to `ownedDataScopeAdminProbes`; the second person is not an admin and must be answered **403**
  (a 404 is a route that is not there and proves nothing). These routes cannot be waived. **An app with
  any admin-only route keeps this test even if all its data is shared**, and the test works with no
  entity registered: leave `ownedDataScopeRows` empty and fill only `ownedDataScopeAdminProbes`.
- **Waivers are narrow.** A row's `Waive` map may excuse only a read (`GET`), never a create,
  update, delete, action or admin-only route; each reason is at least three words and names the
  entity or the route, and no one reason is used on more than two operations; and at most a quarter
  (rounded down, at least one) of the operations the platform requires may be waived.
  The platform counts the quarter and the reuse of a reason across the merged report of every test
  package, while the helper sees one package; so the helper refuses any waiver when `Config.Elsewhere`
  or `Config.Only` is set. Keep the waivers to a package that covers the whole contract.
  The test fails on a waiver that breaks a rule. A waiver is read by a reviewer and is never a way to
  make the test pass.
- **The contract must be readable.** The helper reads `openapi.yaml` with a small reader (the module is
  standard library only) and FAILS the test on YAML it cannot read: anchors, aliases, tags, merge keys,
  tab indentation, a duplicate key, several documents. Write the contract in plain block or flow style.
- **`Elsewhere` is not a waiver.** `Config.Elsewhere` names path prefixes another test package covers (the
  sign-in plugin's own routes, tested beside the code that serves them). The platform merges every
  package's report and still requires each operation; never list a route of your own there. `Config.Only` is the inverse: it covers only the operations under
  its path prefixes (the sign-in plugin's own test uses `/api/auth/`) and leaves the rest to the
  package that owns them.
- **Never skip it.** Do not `t.Skip` it or gate it behind an environment variable; it needs no
  network, no docker and no setting but the optional report path. A test package's tests share one
  report file and merge into it, so more than one `TestOwnedDataScope...` test per package is fine.
- **What it does not prove.** It drives the router in process with two injected identities. It does
  not prove that the deployed sign-in (token checks, the gateway, the authoriser) puts the right
  identity on a request, that a route the contract does not list is scoped, that a store wired
  differently in production scopes the same way, or that a response hides every field of another
  person's record. A plan that declares the data `shared` lowers what the platform requires; this test
  cannot know the plan, so it still covers per-user collections. The platform's checks and review are
  the backstop for the rest.

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
- Declare each key in `albitor.yaml` `config` as a `secret: true` entry with **no `value`** (a value
  on a secret entry is rejected); the person supplies the value through Albitor, never the
  repository. The parameter name is the key, so the read is the prefix plus that key
  (`os.Getenv("ALBITOR_SECRETS_PARAMETER_PREFIX") + "ANTHROPIC_API_KEY"`; the prefix ends in `/`, so
  concatenate). A key not yet supplied reads as not found: that feature is "not configured", not
  an error for the app.
- Read at request time or cache briefly; never log a value or put it in a response.

```yaml
config:
  - key: ANTHROPIC_API_KEY
    secret: true
```

## Fetching a URL a person or a config supplied

`internal/outbound/` is the one way to fetch a URL the app did not write itself: one from a request
body or query, a stored record (a webhook, an avatar or feed URL), or a config value a person set.

- **Any outbound fetch of a request-supplied or user-supplied URL must go through
  `outbound.Client`.** Never fetch a client's URL with a bare `http.Client`, `http.Get` or
  `http.DefaultClient`: a URL such as `http://169.254.169.254/latest/meta-data/` or
  `http://localhost:8080/admin` makes the function fetch from inside its own network (server-side
  request forgery), and the answer comes back to the caller.
- It allows `http` and `https` only, with no credentials in the URL. At dial time it refuses
  loopback, private (RFC 1918, IPv6 unique local), link-local (including the metadata addresses
  `169.254.169.254` and `fd00:ec2::254`), unspecified, multicast and other reserved addresses. The
  check is on the IP actually connected to, so a hostname that resolves to a private address is
  refused, and every redirect is checked again (scheme, allow-list and address). It ignores
  `HTTP_PROXY`, follows at most 5 redirects, times out at 10 seconds and caps the body at 5 MiB.
- **Set `AllowHosts` whenever the hosts are known** (`"api.example.com"`, `"*.example.com"`); an
  empty list means any public host.

```go
fetcher := outbound.New(outbound.Options{AllowHosts: []string{"*.example.com"}}) // build once
resp, err := fetcher.Get(r.Context(), in.URL)
switch {
case errors.Is(err, outbound.ErrBlocked):
	apikit.WriteError(w, http.StatusBadRequest, "that URL is not allowed") // never echo the detail
	return
case err != nil:
	apikit.WriteFailure(w, r, slog.Default(), "fetch url", err) // an opaque 5xx
	return
}
// resp.Status, resp.Header and resp.Body (at most MaxBody bytes) are the answer.
```

`fetcher.Check(url)` validates a URL without fetching (do it when a URL is saved, and fetch through
the client later, which checks again). For a POST or a header use `fetcher.HTTPClient()`: it is the
same guarded `*http.Client`. Do not edit the files under `internal/outbound/` to loosen a check; a
URL that must reach a private address is not a URL from a request.

## Calling an external supplier from the API

Worked example in `examples/api/internal/supplier/` (copy to the same path; standard library only):

- **The supplier's endpoint comes from config, with no default in code.** Declare the base URL as a
  non-secret `config` entry with the value the person gave, read it with `os.Getenv`, and when it
  is empty fail clearly (`SUPPLIER_BASE_URL is not set`). A URL guessed from memory is a wrong host
  that reads like a network fault. The key, if any, is a secret (see above).
- **Fixtures are honest.** A local fixture for the supplier's answer is the supplier's documented
  response, field for field, and nothing else: no field added to make a test or a screen pass. If
  the app needs a value the documented response lacks, report that gap; do not invent the field.
  The example decodes its fixture with `DisallowUnknownFields` against the documented struct, and
  treats a missing documented field as an error rather than a zero value, so a gap shows on the
  first call and not in the deployed walk.

## Calling an LLM supplier from the API

When the app calls a model directly (the Anthropic API, say) rather than through the platform
gateway:

- **The model id comes from config, with no default in code.** Read it from the `config` entry
  (`os.Getenv("LLM_MODEL")`; declare `- key: LLM_MODEL` in `config`, not secret, with the value the person gave) and, when it is empty, fail clearly at start or on first use
  (`LLM_MODEL is not set`). A default in code is a model id from training memory, and a retired id
  is a `404` on every call that reads like a missing key. Never put a model id in a Go constant or
  a default. The id itself comes from the person or from the supplier's live model list at the
  time, never from memory.
- **No sampling parameters.** Do not send `temperature`, `top_p` or `top_k`. Current models reject
  or ignore them (a `400` on some), and the request that works without them is the one to ship.
- **`max_tokens` leaves room for thinking.** Thinking tokens count against `max_tokens`. Set
  `max_tokens` = thinking budget + the expected output, and never below the thinking budget (the
  API refuses `max_tokens` not greater than the budget): with a 4,000 token budget and a 1,000
  token answer, send at least 5,000. With thinking off, size it to the expected output only.

```go
func llmModel(getenv func(string) string) (string, error) {
	m := getenv("LLM_MODEL")
	if m == "" {
		return "", errors.New("LLM_MODEL is not set") // no default: say what is missing
	}
	return m, nil
}

func maxTokens(thinkingBudget, expectedOutput int) int {
	return thinkingBudget + expectedOutput // never less than thinkingBudget
}
```

A call with thinking can take longer than the 30 second request: run it on the queue path
(see Queues), not inside a request.

The request body carries `model`, `max_tokens`, `messages` (and `thinking` when used), and no
`temperature`, `top_p` or `top_k`. Test that the model-unset path fails with a clear error and that
the marshalled request has none of the three sampling fields.

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
