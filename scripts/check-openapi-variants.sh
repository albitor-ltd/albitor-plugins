#!/usr/bin/env bash
# The composer has no slot for the API contract, so each auth plugin that serves GET /api/me ships
# its own api/openapi.yaml (api-go ships the base only for auth-none, via `when`). That is a copy,
# and a copy drifts. This fails when an auth plugin's contract is anything other than api-go's base
# plus its /api/me entry, so a change to the base must be made in every variant.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
base="$root/plugins/api-go/templates/api/openapi.yaml"
fail=0
for variant in "$root"/plugins/auth*/templates/api/openapi.yaml; do
  [[ -e "$variant" ]] || continue
  name="$(basename "$(dirname "$(dirname "$(dirname "$variant")")")")"
  if ! grep -q '^  /api/me:$' "$variant"; then
    echo "error: $name's api/openapi.yaml does not define /api/me" >&2
    fail=1
    continue
  fi
  # Drop the /api/me block: from its path line to the next line at path or comment level.
  if ! diff -u "$base" <(awk '/^  \/api\/me:$/ {skip=1; next} skip && /^(  [^ ]|[^ ])/ {skip=0} !skip' "$variant") >&2; then
    echo "error: $name's api/openapi.yaml differs from api-go's base beyond /api/me" >&2
    fail=1
  fi
done
((fail)) && exit 1
echo "ok: auth plugin contracts are api-go's base plus /api/me"
