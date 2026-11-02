#!/usr/bin/env bash
# Build, vet and test an auth plugin's Go as it composes over api-go: first the seeded files alone
# (they must stand without any example), then with the skill's API examples copied in and the one
# line the skill tells an app to add (registerAuthRoutes in registerIdentityRoutes).
#
#   verify-auth.sh auth-cognito|auth-hosted|auth-none
#
# The real composer lives in albitor-ltd/albitor and is not importable here, so this lays the
# plugin's templates/api tree over api-go's (the paths do not overlap) and renders api-go's
# generated router with the slots filled from the plugin's own manifest, joined as the composer
# joins them (", "): the plugin's api.middleware and api.routes entries are what the tests then
# run behind. scripts/compose-check.sh and tests/compose are the authority on what composes.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
name="${1:?usage: verify-auth.sh auth-cognito|auth-hosted|auth-none}"
case "$name" in auth-cognito | auth-hosted | auth-none) ;; *) echo "unknown auth plugin: $name" >&2; exit 2 ;; esac
apigo="$root/plugins/api-go"
plugin="$root/plugins/$name"
manifest="$plugin/.claude-plugin/plugin.json"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# lay_out <dir> <with-examples>: api-go's tree, the plugin's seeded API, and (optionally) its examples.
lay_out() {
  local dir="$1" examples="$2" middleware routes
  cp -R "$apigo/templates/api/." "$dir/"
  cp -R "$plugin/templates/api/." "$dir/"
  middleware="$(jq -r '(.contributes.slots["api.middleware"] // []) | join(", ")' "$manifest")"
  routes="$(jq -r '(.contributes.slots["api.routes"] // []) | join(", ")' "$manifest")"
  sed -e "s/{{albitor\.slot\.api\.middleware}}/$middleware/" \
    -e "s/{{albitor\.slot\.api\.routes}}/$routes/" \
    "$apigo/templates/api/internal/shell/router.go" >"$dir/internal/shell/router.go"
  if ((examples)); then
    cp -R "$plugin/skills/$name/examples/api/." "$dir/"
    # The join the skill documents: register the sign-in routes from the identity routes.
    sed -i 's|^func registerIdentityRoutes(mux \*http.ServeMux, setup signIn, logger \*slog.Logger) {$|&\n\tregisterAuthRoutes(mux, setup, logger)|' \
      "$dir/internal/shell/identity_verify.go"
    grep -q 'registerAuthRoutes(mux, setup, logger)' "$dir/internal/shell/identity_verify.go" ||
      { echo "error: the registerAuthRoutes join did not apply" >&2; exit 1; }
  fi
  if grep -rn '{{albitor\.slot' "$dir" >/dev/null; then
    echo "error: an unrendered slot placeholder is left in the composed tree" >&2
    exit 1
  fi
}

check_tree() {
  local label="$1"
  (
    cd "$2"
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
  )
  echo "$name: $label API tree builds, vets and passes"
}

mkdir "$work/seed"
lay_out "$work/seed" 0
check_tree "seeded" "$work/seed"

if [[ -d "$plugin/skills/$name/examples/api" ]]; then
  mkdir "$work/examples"
  lay_out "$work/examples" 1
  check_tree "seeded plus examples" "$work/examples"
fi
