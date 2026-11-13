#!/usr/bin/env bash
# A kit's test fixtures are components its tests render to run axe over the pieces: the
# *Fixture.svelte files, PagesShowcase.svelte (the frame the page fixtures render inside),
# ContractProbe.svelte, SignInHarness.svelte, RootClassesProbe.svelte and the Showcase.svelte
# older kits shipped. An app that imports one from a route or a component, or keeps a copy
# outside the fixture folder, ships the fixture (OrgMap, Gistral and Roadkeep each shipped copies,
# albitor-plugins#181). So:
#   - only *.test.ts files and the fixtures themselves may mention them;
#   - a fixture file may live only in .../lib/kit/__fixtures__/ or .../lib/kit/pages/__fixtures__/,
#     the folders `web/knip.json` ignores so the end-of-build prune never targets them (#4846);
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

fixture='(Pages)?Showcase|ContractProbe|SignInHarness|RootClassesProbe|__fixtures__'
fixture_file='^([A-Za-z]+Fixture|(Pages)?Showcase|ContractProbe|SignInHarness|RootClassesProbe)\.svelte$'
fail=0

while IFS= read -r f; do
  base="${f##*/}"
  if [[ "$base" =~ $fixture_file ]]; then
    [[ "$f" == */lib/kit/__fixtures__/"$base" || "$f" == */lib/kit/pages/__fixtures__/"$base" ]] \
      || { echo "error: $f: fixture outside lib/kit/__fixtures__/ or lib/kit/pages/__fixtures__/" >&2; fail=1; }
    continue
  fi
  # Helpers beside the fixtures (dom.ts, nav.ts) are test support too, and tests are tests.
  [[ "$f" == */lib/kit/__fixtures__/* || "$f" == */lib/kit/pages/__fixtures__/* || "$base" == *.test.ts ]] && continue
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
