# Hosted deploy target plugin (`deploy-hosted`)

The `deploy-target@1` plugin for an app on the hosted lane (albitor-ltd/albitor#3173, ADR-0090
section 6), split from the hosted starter tree. A schema-2 plugin: `.claude-plugin/plugin.json`
declares what it provides and contributes, and the platform composer renders it into a new app.

## What it ships

| Repository path in the app | Mode | What |
| --- | --- | --- |
| `.github/workflows/ci.yml` | managed | Advisory manifest check, then build and test of the API and the web app; read-only |

It contributes the `github-actions` entry of the held `dependabot.yml`.

## Behaviour worth knowing

- **No infrastructure in the repo.** What the app needs is declared in `albitor.yaml`; the platform
  compiles it (`core/manifest` `Compile`) and deploys. Nothing is rendered per app.
- **No deploy identity, no secret.** `ci.yml` has `contents: read`, no `id-token`, and reads no
  secret or variable. Triggers are `pull_request`, `push` to main and `workflow_dispatch`.
- **Exactly one deploy target.** This plugin and `deploy-byoa-aws` both provide `deploy-target@1`;
  a profile names one of them.
- **The manifest check is advisory.** The deploy validates `albitor.yaml` with the full validator.
- **Pins are exact.** Every action is a full commit SHA, the Node version is a literal; the Go
  version comes from `api/go.mod`.

## Verification

`tests/compose/deploy_hosted_test.go` (run by `scripts/compose-check.sh`) runs the real
deploy-target@1 conformance suite and workflow lint over the plugin, certifies it in a hosted
profile and composes it, pinning what is written and in which mode.

## Not in this plugin

The deploy caller (`.github/workflows/albitor-deploy.yml`), the deploy context
(`.albitor/deploy.env`) and `.albitor/locked.json` are written by the platform when the repository
is created, not by a plugin: the caller is a job-level `uses` of the platform's reusable pipeline
holding `id-token: write`, which the deploy-target workflow lint refuses.
