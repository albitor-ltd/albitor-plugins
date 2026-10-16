#!/usr/bin/env bash
# Run the real Albitor manifest parser and composer over this repository's schema-2 plugins, and
# optionally build the composed web tree.
#
#   compose-check.sh [--build [--journeys]] <path to an albitor-ltd/albitor checkout>
#
# The parser and composer live in albitor-ltd/albitor (app/core), so a checkout is required. A
# throwaway go.work joins it to tests/compose; nothing in either repository changes.
# --build also runs npm install (the lockfile is the app's, see install_web), check, test and build
# in the composed web/ (needs node >= 22.17; CI pins 24.21.0, as npm 10.9 crashes on a
# lockfile-less install).
# --journeys (with --build) then runs the test-playwright harness over the composed default shell:
# the api-go tree is added, the harness's own unit tests run, and the journeys run against the
# locally booted app in Chromium (needs go; PLAYWRIGHT_INSTALL_FLAGS="--with-deps" in CI).
# The default shell with auth-cognito (web-svelte, kit-shadcn, api-go, test-playwright)
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
mkdir "$tmp/app" "$tmp/online" "$tmp/kit" "$tmp/kit-carbon" "$tmp/auth" "$tmp/offline" "$tmp/byoa"

(cd "$root/tests/compose" && GOWORK="$tmp/go.work" COMPOSE_OUT="$tmp/app" COMPOSE_OUT_ONLINE="$tmp/online" COMPOSE_OUT_OFFLINE="$tmp/offline" COMPOSE_KIT_OUT="$tmp/kit" COMPOSE_KIT_CARBON_OUT="$tmp/kit-carbon" COMPOSE_OUT_AUTH="$tmp/auth" COMPOSE_OUT_BYOA="$tmp/byoa" go test -count=1 ./...)

# Lint the shipped workflows: a step that runs `npm ci` needs a lockfile in its working-directory,
# and the composer drops a plugin's lockfile when the npm merge changes package.json (web/ has none;
# a plugin's own subdirectory, such as web/tests, keeps its own). The plugin must ship that lockfile.
lint_npm_ci() {
  local wf plugin dir status=0
  for wf in "$root"/plugins/*/templates/.github/workflows/*.yml; do
    [[ -e "$wf" ]] || continue
    plugin="${wf%/templates/*}"
    while IFS= read -r dir; do
      if [[ ! -f "$plugin/templates/$dir/package-lock.json" ]]; then
        echo "$wf: npm ci in '$dir' but the plugin ships no $dir/package-lock.json (use npm install)" >&2
        status=1
      fi
    done < <(awk '/run: npm ci/ {ci=1; dir="."; next} ci && /working-directory:/ {dir=$2} ci && /^ *- / {print dir; ci=0} END {if (ci) print dir}' "$wf")
  done
  return $status
}
lint_npm_ci

# The BYOA profile's composed tree: its rendered Terraform validates and plans offline against the
# albitor.yaml the composer wrote (needs terraform; skipped with a note where there is none).
if command -v terraform >/dev/null; then
  "$root/scripts/verify-deploy-byoa-aws.sh" --composed "$tmp/byoa"
else
  echo "terraform not installed: skipping the BYOA composed-tree plan"
fi

# install_web installs the composed web/ in the current directory with `npm install`. The composer
# omits a plugin-declared web/package-lock.json whenever the npm merge changes web/package.json
# (ADR-0090 amendment, albitor#3178), so the lockfile is the app's, not a plugin's, and the shipped
# web-svelte build runs `npm install`: `npm ci` has no lockfile to be reproducible against.
install_web() {
  npm install --no-audit --no-fund
}

if [[ $build -eq 1 ]]; then
  cd "$tmp/app/web"
  install_web
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
  sed -E 's/\{\{albitor\.slot\.[a-z.-]+\}\}//g' "$api/templates/api/internal/shell/router.go" >"$tmp/app/api/internal/shell/router.go"
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

# Each tree below is installed with install_web before it is built. web-svelte with the online
# data layer (data_layer=online):
if [[ $build -eq 1 ]]; then
  cd "$tmp/online/web"
  install_web
  npm run check
  npm test
  npm run build
  test -f build/index.html
  echo "composed web tree with the online data layer builds"
fi

# Same for the offline data layer (data_layer=offline). Its build must also emit the service
# worker into the deploy output: without it the app cannot open offline at all. What the worker
# caches is asserted by shell.test.ts in the app's own tests.
if [[ $build -eq 1 ]]; then
  cd "$tmp/offline/web"
  install_web
  npm run check
  npm test
  npm run build
  test -f build/index.html
  test -f build/service-worker.js
  echo "composed web tree with the offline data layer builds and ships its service worker"
fi

# Same for kit-shadcn (bits-ui, and axe-core and jsdom for its tests). Its tests render the shell
# pieces, run axe over them and audit the colour tokens for WCAG 2.2 AA contrast, so a green run
# here is the kit's accessibility gate.
if [[ $build -eq 1 ]]; then
  cd "$tmp/kit/web"
  install_web
  npm run check
  npm test
  npm run build
  test -f build/index.html
  echo "composed web tree with kit-shadcn builds, and its axe and contrast audits pass"
fi

# Same for kit-carbon (carbon-components-svelte and carbon-icons-svelte, and axe-core and jsdom for
# its tests). Its tests render the shell pieces, run axe over them and audit Carbon's white and
# Gray 100 themes for WCAG 2.2 AA contrast. carbon-components-svelte runs an IBM telemetry
# postinstall; it is switched off here (IBM_TELEMETRY_DISABLED).
if [[ $build -eq 1 ]]; then
  cd "$tmp/kit-carbon/web"
  export IBM_TELEMETRY_DISABLED=true
  install_web
  npm run check
  npm test
  npm run build
  test -f build/index.html
  # The shell must ship Carbon's themes (AppShell imports the stylesheet) and the pinned telemetry
  # package must be the one installed, not whatever carbon-components-svelte's range floats to.
  grep -rlq -e '--cds-text-primary' build/_app || { echo "kit-carbon: Carbon's CSS is not in the build" >&2; exit 1; }
  grep -rlq -e '\[theme=g100\]' build/_app || { echo "kit-carbon: Carbon's g100 theme is not in the build" >&2; exit 1; }
  npm ls @ibm/telemetry-js | grep -q '@ibm/telemetry-js@1.11.0' || { echo "kit-carbon: @ibm/telemetry-js is not the pinned version" >&2; exit 1; }
  echo "composed web tree with kit-carbon builds, and its axe and contrast audits pass"
fi

# The default shell with auth-cognito: the composed API builds, vets and passes (the identity seam
# behind api-go's router, against a fake Cognito and the real local issuer), then the composed web
# tree (kit-shadcn screens, the session, the guard and their tests) checks, tests and builds. Its
# dependencies are installed the way the plugins above are.
if [[ $build -eq 1 ]]; then
  (cd "$tmp/auth/api" && GOWORK=off go vet ./... && GOWORK=off go test ./... -count=1)
  cd "$tmp/auth/web"
  install_web
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
