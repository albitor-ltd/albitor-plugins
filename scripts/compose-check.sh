#!/usr/bin/env bash
# Run the real Albitor manifest parser and composer over this repository's schema-2 plugins, and
# optionally build the composed web tree.
#
#   compose-check.sh [--build] <path to an albitor-ltd/albitor checkout>
#
# The parser and composer live in albitor-ltd/albitor (app/core), so a checkout is required. A
# throwaway go.work joins it to tests/compose; nothing in either repository changes.
# --build also runs npm ci, check, test and build in the composed web/ (needs node >= 22.17).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
build=0
if [[ "${1:-}" == "--build" ]]; then
  build=1
  shift
fi
albitor="${1:?usage: compose-check.sh [--build] <albitor checkout>}"
core="$(cd "$albitor/app/core" 2>/dev/null && pwd)" || { echo "no app/core under $albitor" >&2; exit 2; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
printf 'go 1.26.0\n\nuse (\n\t%s\n\t%s\n)\n' "$root/tests/compose" "$core" >"$tmp/go.work"
mkdir "$tmp/app" "$tmp/online" "$tmp/kit"

(cd "$root/tests/compose" && GOWORK="$tmp/go.work" COMPOSE_OUT="$tmp/app" COMPOSE_OUT_ONLINE="$tmp/online" COMPOSE_KIT_OUT="$tmp/kit" go test -count=1 ./...)

if [[ $build -eq 1 ]]; then
  cd "$tmp/app/web"
  npm ci
  npm run check
  npm test
  npm run build
  test -f build/index.html
  echo "composed web tree builds"
fi

# The composer records a plugin's npm dependencies but does not yet write them into the app's
# package.json (the web-svelte one is app-owned), so install what client-online declares before
# building its composed tree.
if [[ $build -eq 1 ]]; then
  cd "$tmp/online/web"
  mapfile -t deps < <(jq -r '.contributes.dependencies.npm[]' "$root/plugins/client-online/.claude-plugin/plugin.json")
  npm ci
  npm install --save-dev --save-exact --no-audit --no-fund "${deps[@]}"
  npm run check
  npm test
  npm run build
  test -f build/index.html
  echo "composed web tree with client-online builds"
fi

# Same for kit-shadcn (bits-ui, and axe-core and jsdom for its tests). Its tests render the shell
# pieces, run axe over them and audit the colour tokens for WCAG 2.2 AA contrast, so a green run
# here is the kit's accessibility gate.
if [[ $build -eq 1 ]]; then
  cd "$tmp/kit/web"
  mapfile -t deps < <(jq -r '.contributes.dependencies.npm[]' "$root/plugins/kit-shadcn/.claude-plugin/plugin.json")
  npm ci
  npm install --save-dev --save-exact --no-audit --no-fund "${deps[@]}"
  npm run check
  npm test
  npm run build
  test -f build/index.html
  echo "composed web tree with kit-shadcn builds, and its axe and contrast audits pass"
fi
