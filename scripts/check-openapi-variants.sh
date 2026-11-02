#!/usr/bin/env bash
# api-go seeds the base api/openapi.yaml. Each auth plugin that serves GET /api/me carries, as a
# skill example (skills/<plugin>/examples/api/openapi.yaml), the same contract with its /api/me
# (and, for auth-cognito, /api/auth/*) entries added. That is a copy, and a copy drifts. This
# fails when an example contract is anything other than api-go's base plus those entries, so a
# change to the base must be made in every variant.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
base="$root/plugins/api-go/templates/api/openapi.yaml"
fail=0
# Fail closed: every plugin providing the auth socket, other than auth-none (which has no /api/me),
# must ship the example, or its skill has no contract to point an app at.
for manifest in "$root"/plugins/*/.claude-plugin/plugin.json; do
  jq -e '(.provides // []) | index("auth@1")' "$manifest" >/dev/null || continue
  dir="$(dirname "$(dirname "$manifest")")"
  name="$(basename "$dir")"
  [[ "$name" == auth-none ]] && continue
  if [[ ! -f "$dir/skills/$name/examples/api/openapi.yaml" ]]; then
    echo "error: $name provides auth@1 but ships no examples/api/openapi.yaml; its skill has no contract with /api/me to offer" >&2
    fail=1
  fi
done
for variant in "$root"/plugins/auth*/skills/*/examples/api/openapi.yaml; do
  [[ -e "$variant" ]] || continue
  name="$(basename "$(dirname "$(dirname "$(dirname "$(dirname "$(dirname "$variant")")")")")")"
  if ! grep -q '^  /api/me:$' "$variant"; then
    echo "error: $name's api/openapi.yaml does not define /api/me" >&2
    fail=1
    continue
  fi
  # Drop the /api/me and /api/auth/* blocks (the auth-cognito plugin owns those): from a path line to the next line at path or comment level.
  if ! diff -u "$base" <(awk '/^  \/api\/(me|auth\/[^:]*):$/ {skip=1; next} skip && /^(  [^ ]|[^ ])/ {skip=0} !skip' "$variant") >&2; then
    echo "error: $name's api/openapi.yaml differs from api-go's base beyond /api/me and /api/auth/*" >&2
    fail=1
  fi
done
((fail)) && exit 1
echo "ok: auth plugin contracts are api-go's base plus /api/me"
