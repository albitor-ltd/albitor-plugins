#!/usr/bin/env bash
# Prove the pin audit fails closed: build a throwaway copy of the repo's scripts, catalogue and
# plugin manifests, break it three ways, and check the scripts say no.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
fail=0

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
  "$work/r/scripts/build-catalogue.sh" --out "$work/cat.json"
expect_fail "validate-plugins.sh with a schema-1 plugin unpinned" "$work/r/scripts/validate-plugins.sh"

# 2. A pinned plugin flipped to schema 2: it would vanish from the legacy catalogue.
fresh
edit "$work/r/plugins/auth/.claude-plugin/plugin.json" '.schema = 2'
expect_fail "validate-plugins.sh with a pinned plugin flipped to schema 2" "$work/r/scripts/validate-plugins.sh"

exit "$fail"
