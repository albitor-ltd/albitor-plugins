---
name: deploy-hosted
description: Use when an app composed with the deploy-hosted plugin (deploy-target@1) needs a table, a site or API setting, a schedule or a config key, or when its ci workflow fails. There is no infrastructure in the repository; the build changes albitor.yaml, and the seeded ci workflow is the app's to keep green.
metadata:
  capability: deploy-target
---

# Hosted deploy target (deploy-hosted)

This repository holds no infrastructure code. What the app needs from AWS is declared in
`albitor.yaml`, and the platform compiles that file and deploys it. There is nothing to render or
regenerate: change the manifest and the next deploy changes what runs.

## Seeded files (the app's own)

`.github/workflows/ci.yml` is seeded once and is yours from then on. Change it when the app needs
it, and do not weaken the checks it runs.

## Examples (copy, then alter)

Worked examples sit in `examples/` beside this skill, laid out by repository path.

| Example | Copy it when |
| --- | --- |
| `examples/api/internal/gateway/` | The app calls platform services or uses queues (`services.callable` or `queues` in `albitor.yaml`). `cp -r` the directory to `api/internal/gateway`, then edit; a client for the service gateway (`client.go`, SigV4, uploads) and the queues (`queue.go`, `queue_local.go`), and helpers for Transcribe transcripts, Translate byte chunking and PDF page counts (`transcript.go`, `chunk.go`, `pdfpages.go`), with their tests. Standard library only. See the `gateway-services` skill |

An app with neither takes none of it.

## What you may change

| To | Edit |
| --- | --- |
| Add or change a table, a schedule or a config key | the matching block of `albitor.yaml` |
| Call translate, transcribe, textract, bedrock or another AWS AI service | `services.callable` in `albitor.yaml`, then copy the gateway example to `api/internal/gateway`; see the `gateway-services` skill |
| Change how the site or the API is built | `build.static` and `build.api` in `albitor.yaml` |

Do not add Terraform, a deploy workflow, a cloud credential or a role: the platform deploys with
its own identity, and none of them is read.

## The gate

`ci.yml` runs on every pull request and every push to main, with `contents: read` and no secret:

- **manifest** is advisory. It checks the shape every deploy relies on (the version, the top-level
  keys, which build blocks must exist) and prints failures as `{field, code}`. The deploy checks the
  manifest again with the full validator and never trusts this result.
- **api** builds, vets and tests `api/` when it has a `go.mod`.
- **web** installs, checks, tests and builds `web/` when it has a `package.json`.
