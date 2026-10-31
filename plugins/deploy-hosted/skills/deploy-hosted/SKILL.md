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

## Starting points

`.github/workflows/ci.yml` is seeded once and is yours from then on. Change it when the app needs
it, and do not weaken the checks it runs.

## What you may change

| To | Edit |
| --- | --- |
| Add or change a table, a schedule or a config key | the matching block of `albitor.yaml` |
| Call translate, transcribe, textract, bedrock or another AWS AI service | `services.callable` in `albitor.yaml`, then `api/internal/albitorsvc`; see the `gateway-services` skill |
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
