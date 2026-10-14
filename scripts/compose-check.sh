#!/usr/bin/env bash
# Run the real Albitor manifest parser and composer over this repository's schema-2 plugins, and
# optionally build the composed web tree.
#
#   compose-check.sh [--build [--journeys]] <path to an albitor-ltd/albitor checkout>
#
# The parser and composer live in albitor-ltd/albitor (app/core), so a checkout is required. A
# throwaway go.work joins it to tests/compose; nothing in either repository changes.
# --build also runs npm ci, check, test and build in the composed web/ (needs node >= 22.17).
# --journeys (with --build) then runs the test-playwright harness over the composed default shell:
# the api-go tree is added, the harness's own unit tests run, and the journeys run against the
# locally booted app in Chromium (needs go; PLAYWRIGHT_INSTALL_FLAGS="--with-deps" in CI).
# The default shell with auth-cognito (web-svelte, kit-shadcn, client-online, api-go, test-playwright)
# is composed too: --build vets and tests its API and checks, tests and builds its web, and
# --journeys runs the harness plus the sign-in journey against it.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
build=0
journeys=0
if [[ "${1:-}" == "--build" ]]; then
  build=1
  shift
  if [[ "${1:-}" == "--journeys" ]]; then
    journeys=1
    shift
  fi
fi
albitor="${1:?usage: compose-check.sh [--build] <albitor checkout>}"
core="$(cd "$albitor/app/core" 2>/dev/null && pwd)" || { echo "no app/core under $albitor" >&2; exit 2; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
printf 'go 1.26.0\n\nuse (\n\t%s\n\t%s\n)\n' "$root/tests/compose" "$core" >"$tmp/go.work"
mkdir "$tmp/app" "$tmp/online" "$tmp/kit" "$tmp/auth"

(cd "$root/tests/compose" && GOWORK="$tmp/go.work" COMPOSE_OUT="$tmp/app" COMPOSE_OUT_ONLINE="$tmp/online" COMPOSE_KIT_OUT="$tmp/kit" COMPOSE_OUT_AUTH="$tmp/auth" go test -count=1 ./...)

if [[ $build -eq 1 ]]; then
  cd "$tmp/app/web"
  npm ci
  npm run check
  npm test
  npm run build
  test -f build/index.html
  echo "composed web tree builds"
fi

if [[ $journeys -eq 1 ]]; then
  # The api-runtime plugin's tree, composed with every slot empty as scripts/verify-api-go.sh does.
  api="$root/plugins/api-go"
  mkdir -p "$tmp/app/api"
  cp -R "$api/templates/api/." "$tmp/app/api/"
  sed -E 's/\{\{albitor\.slot\.[a-z.-]+\}\}//g' "$api/slots/api/internal/shell/router.go.tmpl" >"$tmp/app/api/internal/shell/router.go"
  cd "$tmp/app/web/tests"
  npm ci
  # shellcheck disable=SC2086 # the flags are a word list on purpose
  npx playwright install ${PLAYWRIGHT_INSTALL_FLAGS:-} chromium
  # The harness's unit tests drive the browser too (a failed sign-in must leave no password).
  npm run unit
  # The temporary go.work above must not capture the composed api/ module.
  GOWORK=off npm test
  echo "journeys pass against the composed default shell"
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

# The default shell with auth-cognito: the composed API builds, vets and passes (the identity seam
# behind api-go's router, against a fake Cognito and the real local issuer), then the composed web
# tree (kit-shadcn screens, the session, the guard and their tests) checks, tests and builds. Its
# dependencies are installed the way the two plugins above are.
if [[ $build -eq 1 ]]; then
  (cd "$tmp/auth/api" && GOWORK=off go vet ./... && GOWORK=off go test ./... -count=1)
  cd "$tmp/auth/web"
  mapfile -t deps < <(jq -r '.contributes.dependencies.npm[]' "$root/plugins/client-online/.claude-plugin/plugin.json" "$root/plugins/kit-shadcn/.claude-plugin/plugin.json")
  npm ci
  npm install --save-dev --save-exact --no-audit --no-fund "${deps[@]}"
  npm run check
  npm test
  npm run build
  test -f build/index.html
  echo "composed default shell with auth-cognito builds, and its session, guard and client tests pass"
fi

# And the journeys over it: a signed-out visitor is sent to /sign-in, the screens render and pass
# axe, and a local identity reaches the app and is known to the API (api-go's local issuer).
if [[ $journeys -eq 1 ]]; then
  cd "$tmp/auth/web/tests"
  npm ci
  # shellcheck disable=SC2086 # the flags are a word list on purpose
  npx playwright install ${PLAYWRIGHT_INSTALL_FLAGS:-} chromium
  GOWORK=off npm test
  echo "journeys pass against the default shell with auth-cognito"
fi
