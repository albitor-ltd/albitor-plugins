# No-auth plugin (`auth-none`)

The `auth@1` plugin for an app with no sign-in (albitor-ltd/albitor#3170, ADR-0090 sections 2 and
13). A schema-2 plugin: `.claude-plugin/plugin.json` declares what it provides and contributes.

## What it ships

| Repository path in the app | Mode | What |
| --- | --- | --- |
| `api/internal/identity/identity.go` | managed | `Owner(r)` always answers `FixedOwner`; the same API as `auth-cognito`'s `identity` package |
| `api/internal/identity/identity_test.go` | managed | Unit tests |

That is all: a stub identity seam with a fixed owner. No screens, no route guard, no middleware, no
routes and no `services.auth`. It requires nothing, so a web-only profile composes it (the files
are written only when `api-go` is the API, `when: api-runtime = api-go`).

## Why it is this small

`auth-none` and `auth-cognito` are interchangeable behind `auth@1`. Handlers call
`identity.Owner(r)` in both, so swapping the plugin changes who the owner is and touches no handler.
A middleware would only re-state `FixedOwner`, and a plugin that fills `api.middleware` has to
require `api-runtime@1`, which a web-only profile does not have.

An app composed with it is open to anyone who can reach it. The skill tells the build to say so in
the delivery record.

## Verification

`scripts/verify-auth.sh auth-none` (run by CI) lays this plugin over `api-go`'s tree and runs
`gofmt`, `go build`, `go vet` and `go test`. `tests/compose` runs the real manifest parser, the
`auth@1` conformance suite and the composer over it, in the default profile and in a web-only one.
