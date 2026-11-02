#!/usr/bin/env bash
# Build, vet and test the deploy-hosted plugin's gateway client, a skill example, as an app would
# use it: copied into api/internal/gateway over api-go's tree.
#
# The real composer lives in albitor-ltd/albitor and is not importable here, so this lays api-go's
# templates/api tree out, copies the example in (`cp -r` into api/internal/gateway, as the skill
# says) and renders api-go's generated router with every slot empty, as verify-data-dynamodb.sh does. scripts/compose-check.sh
# and tests/compose are the authority on what composes.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
apigo="$root/plugins/api-go"
plugin="$root/plugins/deploy-hosted"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cp -R "$apigo/templates/api/." "$work/"
example="$plugin/skills/deploy-hosted/examples/api/internal/gateway"
[[ -d "$example" ]] || { echo "error: $example not found" >&2; exit 1; }
if [[ -e "$plugin/templates/api" ]]; then
  echo "error: deploy-hosted seeds nothing under api/; the gateway client is a skill example" >&2
  exit 1
fi
cp -R "$example" "$work/internal/gateway"
sed -E 's/\{\{albitor\.slot\.[a-z.-]+\}\}//g' \
  "$apigo/templates/api/internal/shell/router.go" >"$work/internal/shell/router.go"

cd "$work"
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  echo "error: not gofmt-clean:" >&2
  echo "$unformatted" >&2
  exit 1
fi

# The module stays stdlib only: the plugin must not need a dependency api-go's go.mod lacks.
if grep -q '^require' go.mod; then
  echo "error: go.mod gained a dependency; the gateway client must stay stdlib-only" >&2
  exit 1
fi

go build ./...
go vet ./...
go test ./internal/gateway/ -count=1 -race
