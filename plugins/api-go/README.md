# Go API runtime plugin (`api-go`)

The `api-runtime@1` plugin for Go (albitor-ltd/albitor#3168, ADR-0090 sections 5 and 13), split from
the starter's `_api`. It is a schema-2 plugin: `.claude-plugin/plugin.json` declares what it
provides, requires and contributes, and the platform composer renders it into a new app.

## What it ships

| Repository path in the app | Mode | What |
| --- | --- | --- |
| `api/go.mod`, `api/main.go` | managed | Module `webapp` (stdlib only) and the process entry point |
| `api/internal/shell/shell.go`, `boot.go` | managed | Health route, JSON 404, middleware chain, request logging, local issuer wiring |
| `api/internal/shell/router.go` | generated | Places the `api.middleware` and `api.routes` slots; designated by `slots/slots.json` |
| `api/internal/apikit/**` | managed | JSON writers, the `Page` envelope, cursors, `ServeList`, correlation ids |
| `api/internal/localauth/**` | managed | The Go side of the local run contract: the in-process test issuer |
| `api/internal/app/routes.go` | example | `Routes(*http.ServeMux)`, where product handlers are registered |
| `api/openapi.yaml`, `api/README.md`, `api/.gitignore` | example | The API contract and notes |

It contributes the `gomod` entry for `/api` to `repo.dependabot`, the `build.api` and `http` blocks
of `albitor.yaml`, and the `api-go` skill. It requires `deploy-target@1` because that socket hosts
`repo.dependabot`. There is no items demo: the store, secrets and auth belong to the data-store,
object-store and auth plugins.

## Layout of this plugin

```text
.claude-plugin/plugin.json   schema-2 manifest
templates/api/...            files the composer copies; a contributed file at repo path P is
                             read from templates/P, with {{albitor.*}} tokens substituted
slots/slots.json             the generated slot files this plugin designates (path, syntax, source)
slots/api/internal/shell/    the slot template bodies, with {{albitor.slot.<name>}} placeholders
skills/api-go/SKILL.md       how to add an endpoint on this shell
```

`templates/api` is a Go module on its own, but not a complete one until the router is generated.
`scripts/verify-api-go.sh` (run by CI) renders the router with every slot empty and runs
`go build`, `go vet` and `go test` on the result, and fails if the rendered router (empty or
populated slots) is not gofmt-clean. The template puts each placeholder inside a call's
parentheses (`middlewareOf(...)`, `routesOf(...)`) so it is clean either way.
`scripts/test-catalogue-guards.sh` proves the legacy pin audit fails closed.

## The slot contract

A fragment is an identifier resolved in package `internal/shell`. `api.middleware` entries are
`func(http.Handler) http.Handler`, outermost first, inside request logging; `api.routes` entries
are `func(*http.ServeMux)`, registered before the app's `Routes`. A contributing plugin ships a
managed file in `api/internal/shell/` that declares the symbol.
