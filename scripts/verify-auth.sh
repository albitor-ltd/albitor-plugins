#!/usr/bin/env bash
# Build, vet and test an auth plugin's Go as it composes over api-go.
#
#   verify-auth.sh auth-cognito|auth-none
#
# The real composer lives in albitor-ltd/albitor and is not importable here, so this lays the
# plugin's templates/api tree over api-go's (the paths do not overlap) and renders api-go's
# generated router with the slots filled from the plugin's own manifest, joined as the composer
# joins them (", "): the plugin's api.middleware and api.routes entries are what the tests then
# run behind. scripts/compose-check.sh and tests/compose are the authority on what composes.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
name="${1:?usage: verify-auth.sh auth-cognito|auth-none}"
case "$name" in auth-cognito | auth-none) ;; *) echo "unknown auth plugin: $name" >&2; exit 2 ;; esac
apigo="$root/plugins/api-go"
plugin="$root/plugins/$name"
manifest="$plugin/.claude-plugin/plugin.json"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cp -R "$apigo/templates/api/." "$work/"
cp -R "$plugin/templates/api/." "$work/"

middleware="$(jq -r '(.contributes.slots["api.middleware"] // []) | join(", ")' "$manifest")"
routes="$(jq -r '(.contributes.slots["api.routes"] // []) | join(", ")' "$manifest")"
sed -e "s/{{albitor\.slot\.api\.middleware}}/$middleware/" \
  -e "s/{{albitor\.slot\.api\.routes}}/$routes/" \
  "$apigo/templates/api/internal/shell/router.go" >"$work/internal/shell/router.go"

if grep -rn '{{albitor\.slot' "$work" >/dev/null; then
  echo "error: an unrendered slot placeholder is left in the composed tree" >&2
  exit 1
fi

cd "$work"
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  echo "error: not gofmt-clean:" >&2
  echo "$unformatted" >&2
  exit 1
fi

# The plugin may not add a dependency to api/go.mod, which api-go owns (ADR-0090 amendment, item 3).
if grep -q '^require' go.mod; then
  echo "error: go.mod gained a dependency; the auth plugins must stay stdlib-only" >&2
  exit 1
fi

go build ./...
go vet ./...
go test ./... -count=1
echo "$name: composed API tree builds, vets and passes"
