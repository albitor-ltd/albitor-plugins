#!/usr/bin/env bash
# Build, vet and test the object-s3 plugin's storage example as an app would use it: copied into
# api/internal/objects and api/internal/app over api-go's tree, with auth-hosted's identity package
# (the one identity API a handler uses).
#
# The real composer lives in albitor-ltd/albitor and is not importable here, so this lays api-go's
# templates/api tree out, copies the example in (as the skill says) and renders api-go's generated
# router with every slot empty, as verify-deploy-hosted.sh does. scripts/compose-check.sh and
# tests/compose are the authority on what composes.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
apigo="$root/plugins/api-go"
plugin="$root/plugins/object-s3"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

if [[ -e "$plugin/templates" ]]; then
  echo "error: object-s3 seeds nothing; the storage code is a skill example" >&2
  exit 1
fi
example="$plugin/skills/object-s3/examples/api/internal"
[[ -d "$example/objects" && -d "$example/app" ]] || { echo "error: $example not found" >&2; exit 1; }

cp -R "$apigo/templates/api/." "$work/"
cp -R "$root/plugins/auth-hosted/templates/api/internal/identity" "$work/internal/identity"
cp -R "$example/objects" "$work/internal/objects"
cp "$example"/app/files.go "$example"/app/files_test.go "$work/internal/app/"
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
  echo "error: go.mod gained a dependency; the storage example must stay stdlib-only" >&2
  exit 1
fi

go build ./...
go vet ./...
go test ./internal/objects/ ./internal/app/ -count=1 -race
