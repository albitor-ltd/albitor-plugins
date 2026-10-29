#!/usr/bin/env bash
# Build, vet and test the deploy-hosted plugin's Go template (api/internal/albitorsvc) as it
# composes over api-go.
#
# The real composer lives in albitor-ltd/albitor and is not importable here, so this lays the two
# plugins' templates/api trees over each other (the paths do not overlap) and renders api-go's
# generated router with every slot empty, as verify-data-dynamodb.sh does. scripts/compose-check.sh
# and tests/compose are the authority on what composes.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
apigo="$root/plugins/api-go"
plugin="$root/plugins/deploy-hosted"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cp -R "$apigo/templates/api/." "$work/"
cp -R "$plugin/templates/api/." "$work/"
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
  echo "error: go.mod gained a dependency; the albitorsvc client must stay stdlib-only" >&2
  exit 1
fi

go build ./...
go vet ./...
go test ./internal/albitorsvc/ -count=1 -race
