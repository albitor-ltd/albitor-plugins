#!/usr/bin/env bash
# The Playwright the test-playwright plugin pins must equal the Playwright the Albitor runner image
# bakes a browser for (albitor-ltd/albitor#3895). Offline, a spec on any other version cannot launch
# a browser: Playwright looks for the chromium revision of ITS own release and the image holds
# only its own.
#
# runner-image.json records the image's version (container/runner/axe-scan/package.json in
# albitor-ltd/albitor). This script fails when the plugin's package.json or lockfile disagree with
# it, so a routine dependency bump of the plugin cannot silently outrun the image. To move the pin:
# publish a runner image on the new version, then change this record and the plugin together.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tests="$root/plugins/test-playwright/templates/web/tests"
command -v jq >/dev/null || { echo "jq is required" >&2; exit 2; }

want="$(jq -er '.playwright' "$root/runner-image.json")"
fail=0
check() { # label actual
  if [[ "$2" != "$want" ]]; then
    echo "error: $1 is $2 but the runner image bakes Playwright $want (runner-image.json)" >&2
    fail=1
  fi
}

check "test-playwright @playwright/test" "$(jq -er '.devDependencies["@playwright/test"]' "$tests/package.json")"
check "test-playwright playwright-core" "$(jq -er '.devDependencies["playwright-core"]' "$tests/package.json")"
for pkg in @playwright/test playwright playwright-core; do
  check "test-playwright lockfile $pkg" "$(jq -er --arg p "node_modules/$pkg" '.packages[$p].version' "$tests/package-lock.json")"
done

if ((fail)); then
  echo "Bump the runner image first (albitor container/runner/axe-scan), then runner-image.json and the plugin together." >&2
  exit 1
fi
echo "ok: test-playwright and the runner image both use Playwright $want"
