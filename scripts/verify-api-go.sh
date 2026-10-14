#!/usr/bin/env bash
# Build, vet and test the api-go plugin's tree as it composes with no other plugin.
#
# The real composer lives in albitor-ltd/albitor and is not importable here, so this renders the
# one thing it adds, the generated router (slots/api/internal/shell/router.go.tmpl), with every
# slot empty, over a copy of templates/api. The composer's golden test (in the albitor repo) is the
# authority on what composes; this is the plugin's own gate that its Go compiles and passes.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
plugin="$root/plugins/api-go"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cp -R "$plugin/templates/api/." "$work/"
sed -E 's/\{\{albitor\.slot\.[a-z.-]+\}\}//g' \
  "$plugin/slots/api/internal/shell/router.go.tmpl" >"$work/internal/shell/router.go"

if grep -rn '{{albitor\.slot' "$work" >/dev/null; then
  echo "error: an unrendered slot placeholder is left in the composed tree" >&2
  exit 1
fi

cd "$work"
# The generated router is not gofmt-formatted (empty slots leave padding), so format the tree
# before the check applies to hand-written files only.
gofmt -w internal/shell/router.go
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  echo "error: not gofmt-clean:" >&2
  echo "$unformatted" >&2
  exit 1
fi
go build ./...
go vet ./...
go test ./... -count=1
