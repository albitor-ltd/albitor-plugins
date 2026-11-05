#!/usr/bin/env bash
# Showcase.svelte, PagesShowcase.svelte and ContractProbe.svelte are test fixtures: the kit's tests
# render them to run axe over every piece. An app that imports one from a route or a component, or
# keeps a copy outside the kit folder, ships the fixture (OrgMap, Gistral and Roadkeep each shipped
# copies, albitor-plugins#181). So:
#   - only *.test.ts files and the fixtures themselves may mention them;
#   - a fixture file may live only at .../lib/kit/<name> (Showcase, ContractProbe) or
#     .../lib/kit/pages/PagesShowcase.svelte, never beside a route or elsewhere;
#   - no skill may tell an agent to start from one.
#
#   check-kit-fixtures.sh <dir>...
#
# compose-check.sh runs it over each composed web/src; `just ci` over plugins/.
set -euo pipefail

(($#)) || { echo "usage: check-kit-fixtures.sh <dir>..." >&2; exit 2; }
for d in "$@"; do
  [[ -d "$d" ]] || { echo "check-kit-fixtures: no such directory: $d" >&2; exit 2; }
done

fixture='(Pages)?Showcase|ContractProbe'
fail=0

while IFS= read -r f; do
  base="${f##*/}"
  case "$base" in
    Showcase.svelte | ContractProbe.svelte)
      [[ "$f" == */lib/kit/"$base" ]] || { echo "error: $f: fixture outside lib/kit/" >&2; fail=1; }
      continue ;;
    PagesShowcase.svelte)
      [[ "$f" == */lib/kit/pages/"$base" ]] || { echo "error: $f: fixture outside lib/kit/pages/" >&2; fail=1; }
      continue ;;
    *.test.ts) continue ;;
  esac
  if grep -qE "$fixture" "$f"; then
    echo "error: $f references a test fixture ($(grep -oE "$fixture" "$f" | sort -u | paste -sd, -)); only *.test.ts may" >&2
    fail=1
  fi
done < <(find "$@" -type f \( -name '*.svelte' -o -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.mjs' \) \
  -not -path '*/node_modules/*' -not -path '*/.svelte-kit/*' -not -path '*/dist/*')

while IFS= read -r line; do
  [[ "$line" == *fixture* ]] && continue
  echo "error: tells an agent to start from or copy a test fixture: $line" >&2
  fail=1
done < <(grep -rhiE "((start|begin) (from|with)|copy)[^.]*(Pages)?Showcase" "$@" --include='*.md' 2>/dev/null || true)

exit "$fail"
