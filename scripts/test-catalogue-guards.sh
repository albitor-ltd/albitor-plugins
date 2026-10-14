#!/usr/bin/env bash
# Prove the pin audit fails closed: build a throwaway copy of the repo's scripts, catalogue and
# plugin manifests, break it several ways, and check the scripts say no.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
fail=0
# The throwaway copy has no git history: resolve the release tags in the real checkout.
export CATALOGUE_TAG_REPO="$root"

fresh() {
  rm -rf "$work/r"
  mkdir -p "$work/r/plugins"
  cp -R "$root/scripts" "$root/catalogue" "$root/.claude-plugin" "$work/r/"
  for d in "$root"/plugins/*/; do
    mkdir -p "$work/r/plugins/$(basename "$d")/.claude-plugin"
    cp "$d.claude-plugin/plugin.json" "$work/r/plugins/$(basename "$d")/.claude-plugin/"
  done
}

expect_fail() { # description, command...
  local what="$1"
  shift
  if "$@" >"$work/out" 2>&1; then
    echo "FAIL: $what was accepted" >&2
    fail=1
  else
    echo "ok: $what is rejected"
  fi
}

# edit FILE JQ-FILTER rewrites a JSON file in place.
edit() {
  jq "$2" "$1" >"$work/tmp" && mv "$work/tmp" "$1"
}

fresh
"$work/r/scripts/validate-plugins.sh" >/dev/null
"$work/r/scripts/build-catalogue.sh" --check >/dev/null

# 1. A schema-1 plugin whose pin is gone: legacy mode must error, not omit it.
fresh
edit "$work/r/catalogue/legacy-pins.json" 'del(.auth)'
expect_fail "build-catalogue.sh (legacy) with a schema-1 plugin unpinned" \
  "$work/r/scripts/build-catalogue.sh" --mode legacy --out "$work/cat.json"
expect_fail "validate-plugins.sh with a schema-1 plugin unpinned" "$work/r/scripts/validate-plugins.sh" --mode legacy

# 2. A pinned plugin flipped to schema 2: it would vanish from the legacy catalogue.
fresh
edit "$work/r/plugins/auth/.claude-plugin/plugin.json" '.schema = 2'
expect_fail "validate-plugins.sh with a pinned plugin flipped to schema 2" "$work/r/scripts/validate-plugins.sh" --mode legacy

# 3. A plugin with no version: the generator must not emit <name>/vnull.
fresh
edit "$work/r/plugins/auth/.claude-plugin/plugin.json" 'del(.version)'
expect_fail "build-catalogue.sh with a plugin that has no version" \
  "$work/r/scripts/build-catalogue.sh" --out "$work/cat.json"

# 4. A plugin tag that does not exist: the catalogue must not pin it.
fresh
edit "$work/r/plugins/auth/.claude-plugin/plugin.json" '.version = "9.9.9"'
expect_fail "build-catalogue.sh pinning a tag that does not exist" \
  "$work/r/scripts/build-catalogue.sh" --out "$work/cat.json"

# 5. Catalogue metadata drifting from plugin.json.
fresh
edit "$work/r/catalogue/metadata.json" '.auth.description = "drifted"'
expect_fail "validate-plugins.sh with a drifted catalogue description" "$work/r/scripts/validate-plugins.sh"

# 6. Monorepo mode (the default) needs no legacy pin.
fresh
edit "$work/r/catalogue/legacy-pins.json" 'del(.auth)'
"$work/r/scripts/validate-plugins.sh" >/dev/null || { echo "FAIL: monorepo validation demanded a legacy pin" >&2; fail=1; }

# 7. An unreleased plugin (no tag yet) is left out of the catalogue instead of failing it, and
#    nothing else changes.
fresh
edit "$work/r/plugins/auth/.claude-plugin/plugin.json" '.version = "9.9.9"'
edit "$work/r/catalogue/metadata.json" '.auth.unreleased = true'
"$work/r/scripts/build-catalogue.sh" --out "$work/cat.json" >/dev/null || { echo "FAIL: an unreleased plugin broke generation" >&2; fail=1; }
if jq -e '.plugins[] | select(.name == "auth")' "$work/cat.json" >/dev/null; then
  echo "FAIL: an unreleased plugin is in the catalogue" >&2
  fail=1
else
  echo "ok: an unreleased plugin is left out"
fi
if ! jq -e '.plugins[] | select(.name == "gds")' "$work/cat.json" >/dev/null; then
  echo "FAIL: the unreleased flag dropped a released plugin" >&2
  fail=1
fi

# 8. The flag cannot hide a plugin whose tag exists.
fresh
edit "$work/r/catalogue/metadata.json" '.auth.unreleased = true'
expect_fail "build-catalogue.sh with an unreleased flag on a tagged plugin" \
  "$work/r/scripts/build-catalogue.sh" --out "$work/cat.json"

exit "$fail"
