# DynamoDB data store plugin (`data-dynamodb`)

The `data-store@1` plugin (albitor-ltd/albitor#3169, ADR-0090 sections 6 and 13), split from the
starter's `_api/server/store*.go`. A schema-2 plugin: `.claude-plugin/plugin.json` declares what it
provides, requires and contributes, and the platform composer renders it into a new app.

## What it ships

| Repository path in the app | Mode | What |
| --- | --- | --- |
| `api/internal/store/store.go` | managed | `Store`, `Record`, `Query`, the errors, `Key`, validation, cursors |
| `api/internal/store/memory.go` | managed | The in-memory fake for a local run and tests |
| `api/internal/store/dynamodb.go`, `sigv4.go` | managed | The DynamoDB adapter: JSON API over `net/http`, SigV4, retries |
| `api/internal/store/open.go` | managed | `Open`: picks the adapter from the environment, never falls back to memory |
| `api/internal/store/*_test.go` | managed | Unit tests and the in-memory conformance run |
| `api/internal/store/storetest/**` | managed | The data-store@1 conformance suite |

It contributes the `data` table to `services.tables` in `albitor.yaml` (string `pk` and `sk`) and
the `data-dynamodb` skill. It requires `api-runtime@1` (the module, `apikit` paging and the local
run contract). It contributes no Terraform: the deploy target realises the table, and hands the
function its name as `ALBITOR_TABLE_DATA`.

The module stays stdlib-only. The adapter signs its own requests (SigV4) rather than importing the
AWS SDK, because the plugin may not add a dependency to `api/go.mod`, which `api-go` owns
(ADR-0090 amendment 3: logic ships as managed files for now). Moving the adapter into a versioned
module later is an ordinary plugin major.

## Behaviour worth knowing

- One table, `pk` and `sk` strings, the record's JSON in `data`, a revision in `rev`.
- `Create` is conditional (`ErrExists`); `Replace` and `Delete` are guarded by the revision the
  caller read (`ErrConflict`, `ErrNotFound`).
- `Query` is one partition, an optional literal sort-key prefix, either direction, with an opaque
  cursor bound to the partition and prefix. It reads `Limit+1` rows so a cursor means more
  follows, and keeps reading when DynamoDB cuts a response short at 1 MB.
- Reads are strongly consistent.
- Credentials are environment variables only (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`,
  `AWS_SESSION_TOKEN`, `AWS_REGION`), which is what Lambda sets for the execution role. There is
  no profile, file or metadata-service lookup. The HTTP client ignores proxy environment variables.
- Writes are not retried on a 5xx (it may have applied); throttling is retried, and so is a read's 5xx.
- On a deployed function `ALBITOR_DYNAMODB_ENDPOINT` is ignored.
- `ALBITOR_DYNAMODB_ENDPOINT` points the adapter at DynamoDB Local and accepts loopback hosts
  only: a signed request is never sent to a host the platform did not name. Responses are capped,
  redirects are not followed, and DynamoDB error messages (which can name roles and accounts) are
  never put in an error.

## Verification

`scripts/verify-data-dynamodb.sh` (run by CI) lays this plugin over `api-go`'s tree, renders
`api-go`'s router with empty slots, and runs `gofmt`, `go build`, `go vet` and `go test`, then
runs the conformance suite against DynamoDB Local (`amazon/dynamodb-local`, started and removed by
the script, or `DYNAMODB_LOCAL_ENDPOINT`). The same suite runs against the in-memory fake in the
managed tree, so an app's own CI proves the fake without Docker. `conformance/` holds the one test
that needs DynamoDB Local; it is not shipped to apps, and it fails rather than skips when no
endpoint is set.
`tests/compose` runs the real manifest parser and composer over this plugin and `api-go`.
