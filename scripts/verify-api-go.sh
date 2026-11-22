#!/usr/bin/env bash
# Build, vet and test the api-go plugin's tree as it composes with no other plugin.
#
# The real composer lives in albitor-ltd/albitor and is not importable here, so this renders the
# one thing it adds, the generated router (templates/api/internal/shell/router.go), with every
# slot empty, over a copy of templates/api. The composer's golden test (in the albitor repo) is the
# authority on what composes; this is the plugin's own gate that its Go compiles and passes.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
plugin="$root/plugins/api-go"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cp -R "$plugin/templates/api/." "$work/"
sed -E 's/\{\{albitor\.slot\.[a-z.-]+\}\}//g' \
  "$plugin/templates/api/internal/shell/router.go" >"$work/internal/shell/router.go"

if grep -rn '{{albitor\.slot' "$work" >/dev/null; then
  echo "error: an unrendered slot placeholder is left in the composed tree" >&2
  exit 1
fi

# The helper stamps the kit release into its report (ownedscope/provenance.go): it must be this one.
kit="$(jq -r .version "$plugin/.claude-plugin/plugin.json")"
if ! grep -q "^const KitVersion = \"$kit\"\$" "$plugin/templates/api/internal/ownedscope/provenance.go"; then
  echo "error: ownedscope KitVersion is not the plugin's version ($kit)" >&2
  exit 1
fi

cd "$work"
# The generated router must be gofmt-clean as composed, empty slots included: no formatting here.
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  echo "error: not gofmt-clean:" >&2
  echo "$unformatted" >&2
  exit 1
fi

# The same template with populated slots (joined as the composer joins them, ", ") is clean too.
populated="$(mktemp -d)"
sed -e 's/{{albitor\.slot\.api\.middleware}}/mwA, mwB/' -e 's/{{albitor\.slot\.api\.routes}}/routesA/' \
  "$plugin/templates/api/internal/shell/router.go" >"$populated/router.go"
if [[ -n "$(gofmt -l "$populated/router.go")" ]]; then
  echo "error: the router template is not gofmt-clean with populated slots" >&2
  rm -rf "$populated"
  exit 1
fi
rm -rf "$populated"

# The feature example (one Go package per feature) wired in as the skill says: an import line and a
# Register line appended to registry.go. The result must build, mount and pass, and registry.go must
# still hold no function line, which is what lets the platform's rebase resolver union two appends.
if grep -nE '(^|[[:space:]])func[[:space:]]|=>' internal/app/registry.go; then
  echo "error: internal/app/registry.go has code; it must be the imports and the registrars list only" >&2
  exit 1
fi
cp -R "$plugin/skills/api-go/examples/api/internal/widgets" internal/widgets
sed -i \
  -e 's|^// one import line per feature, appended below$|&\nimport "webapp/internal/widgets"|' \
  -e 's|^\t// one line per feature, appended below$|&\n\twidgets.Register,|' \
  internal/app/registry.go
grep -q '^import "webapp/internal/widgets"$' internal/app/registry.go || { echo "error: the registry import anchor moved" >&2; exit 1; }
grep -q '^	widgets.Register,$' internal/app/registry.go || { echo "error: the registry list anchor moved" >&2; exit 1; }
if [[ -n "$(gofmt -l .)" ]]; then
  echo "error: not gofmt-clean with the widgets feature wired in" >&2
  exit 1
fi
# The seeded tree with the feature example (not the supplier example below) must pass the lint a delivered app runs (the
# platform's lint gate: bare `golangci-lint run ./...`, default linters). The vendored helper must pass it as it is, or a foundation run "fixes" it and moves the
# source digest the platform checks (albitor-ltd/albitor#4447).
golangci-lint run --allow-parallel-runners ./...

# The supplier example, copied into the composed tree as the skill says, builds and passes too.
cp -R "$plugin/skills/api-go/examples/api/internal/supplier" internal/supplier

go build ./...
go vet ./...
go test ./... -count=1
