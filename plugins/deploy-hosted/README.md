# Hosted deploy target plugin (`deploy-hosted`)

The `deploy-target@1` plugin for an app on the hosted lane (albitor-ltd/albitor#3173, ADR-0090
section 6), split from the hosted starter tree. A schema-2 plugin: `.claude-plugin/plugin.json`
declares what it provides and contributes, and the platform composer renders it into a new app.

## What it ships

| Repository path in the app | Mode | What |
| --- | --- | --- |
| `.github/workflows/ci.yml` | managed | Advisory manifest check, then build and test of the API and the web app; read-only |
| `api/internal/albitorsvc/**` | managed | `albitorsvc`, the thin Go client for the Albitor service gateway, with its tests (below) |

It contributes the `github-actions` entry of the held `dependabot.yml`.

## Behaviour worth knowing

- **No infrastructure in the repo.** What the app needs is declared in `albitor.yaml`; the platform
  compiles it (`core/manifest` `Compile`) and deploys. Nothing is rendered per app.
- **No deploy identity, no secret.** `ci.yml` has `contents: read`, no `id-token`, and reads no
  secret or variable. Triggers are `pull_request`, `push` to main and `workflow_dispatch`.
- **The platform seeds the accounts.** The demo and verifier accounts are created in the preview's
  user pool by the platform's pinned pipeline on each preview deploy (Albitor's ADR-0071, 2026-10-03
  entry), under a cell role, from passwords only Albitor writes. Nothing in this plugin or the
  repository does, and a live deploy seeds none.
- **Exactly one deploy target.** This plugin and `deploy-byoa-aws` both provide `deploy-target@1`;
  a profile names one of them.
- **The manifest check is advisory.** The deploy validates `albitor.yaml` with the full validator.
- **Pins are exact.** Every action is a full commit SHA, the Node version is a literal; the Go
  version comes from `api/go.mod`.

## The service gateway client (`albitorsvc`)

albitor-ltd/albitor#4005, ADR-0099 section 3.1. An app declares `services.callable` in
`albitor.yaml` (`renders` lists it, so the composer accepts the block) and calls AWS AI services
through the gateway, keyless and credit-metered, with `api/internal/albitorsvc`:

- `POST <ALBITOR_SERVICES_URL>/v1/<service>/<Operation>`, the AWS API's JSON body, SigV4 for service
  `lambda` with the function role's credentials from the environment. Standard library only, so it
  adds nothing to `api/go.mod`; it does not pull in `aws-sdk-go-v2` (the issue's "reuse its types"
  becomes: the app writes structs to AWS's documented field names).
- Typed errors: `402 credits_exhausted` and `cap_reached`, any `403`, and `429` match
  `errors.Is` against `ErrCreditsExhausted`, `ErrCapReached`, `ErrForbidden` and `ErrRateLimited`;
  `Unavailable(err)` groups the 402s and 403s as "feature unavailable". AWS's message never arrives.
- Only a `429` is retried (it is refused before the call runs), with full-jitter capped
  exponential backoff never sooner than `Retry-After`; a `Retry-After` over the 5 second cap is not waited for and the 429 returns at once. An idempotency key must match `^[A-Za-z0-9_-]{8,64}$`.
- `CreateUploadSlot` and `Upload` for the gateway's presigned POST slots; the slot is never logged
  or put in an error.
- No redirect is followed and the gateway URL must be https (loopback http for tests).

The `gateway-services` skill carries the build guidance: declare `services.callable`, put
gateway-backed features behind sign-in or per-end-user rate limits by default (ADR-0099 section 12,
Denial of wallet), and treat `402` as "feature unavailable".

## Verification

`scripts/verify-deploy-hosted.sh` (the `deploy-hosted` recipe of `just ci`) lays the template over
api-go's, then builds, vets and tests `albitorsvc` with `-race` and fails if `go.mod` gains a
dependency.

`tests/compose/deploy_hosted_test.go` (run by `scripts/compose-check.sh`) runs the real
deploy-target@1 conformance suite and workflow lint over the plugin, certifies it in a hosted
profile and composes it, pinning what is written and in which mode.

## Not in this plugin

The deploy caller (`.github/workflows/albitor-deploy.yml`), the deploy context
(`.albitor/deploy.env`) and `.albitor/locked.json` are written by the platform when the repository
is created, not by a plugin: the caller is a job-level `uses` of the platform's reusable pipeline
holding `id-token: write`, which the deploy-target workflow lint refuses.
