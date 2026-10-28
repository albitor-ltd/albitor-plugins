# Local merge gate: `just ci` is the only gate; there is no hosted CI workflow (#98). Needs
# shellcheck, jq, go, terraform and Docker (data-dynamodb starts amazon/dynamodb-local). Needs
# network for Go modules, the Terraform registry and the dynamodb-local image.

default:
    @just --list

ci: manifests catalogue lint api-go data-dynamodb auth deploy-byoa-aws

manifests:
    scripts/validate-plugins.sh
    scripts/check-runner-playwright.sh
    scripts/test-catalogue-guards.sh
    scripts/validate-preview-tokens.sh
    scripts/test-preview-tokens.sh

catalogue: catalogue-check catalogue-legacy

catalogue-check:
    scripts/build-catalogue.sh --check

catalogue-legacy:
    #!/usr/bin/env bash
    set -euo pipefail
    out="$(mktemp)"; trap 'rm -f "$out"' EXIT
    scripts/build-catalogue.sh --mode legacy --out "$out"
    jq -e '.plugins | length > 0' "$out"

lint:
    find scripts plugins -name '*.sh' -print0 | xargs -0 shellcheck
    find . -name '*.json' -not -path './.git/*' -print0 | xargs -0 -n1 jq -e . >/dev/null

api-go:
    scripts/verify-api-go.sh

data-dynamodb:
    scripts/verify-data-dynamodb.sh

auth:
    scripts/verify-auth.sh auth-cognito
    scripts/verify-auth.sh auth-none
    scripts/verify-auth.sh auth-hosted

deploy-byoa-aws:
    scripts/verify-deploy-byoa-aws.sh

# Not part of `ci`, but REQUIRED before merging a change to plugin templates, manifests or the
# composer contract: needs a checkout of the private albitor-ltd/albitor repo (app/core) and
# Node >= 22.17 plus npm (and Playwright with --journeys), which this gate does not require.
# Usage: just compose ~/albitor/albitor
compose albitor:
    scripts/compose-check.sh --build {{albitor}}
