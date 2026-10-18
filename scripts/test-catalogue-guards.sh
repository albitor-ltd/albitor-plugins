#!/usr/bin/env bash
# Prove the pin audit fails closed: build a throwaway copy of the repo's scripts, catalogue and
# plugin manifests, break it three ways, and check the scripts say no.
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

# 7. A new plugin (not yet in the committed catalogue, no tag) is left out with a note.
fresh
mkdir -p "$work/r/plugins/brand-new/.claude-plugin"
jq '.name = "brand-new" | .version = "0.0.1"' "$root/plugins/api-go/.claude-plugin/plugin.json" >"$work/r/plugins/brand-new/.claude-plugin/plugin.json"
edit "$work/r/catalogue/metadata.json" '.["brand-new"] = .["api-go"]'
"$work/r/scripts/build-catalogue.sh" --out "$work/new.json" >"$work/out" 2>&1 || { echo "FAIL: a new untagged plugin broke generation" >&2; fail=1; }
grep -q 'brand-new' "$work/out" || { echo "FAIL: no note for the untagged new plugin" >&2; fail=1; }
! jq -e 'any(.plugins[]; .name == "brand-new")' "$work/new.json" >/dev/null || { echo "FAIL: the untagged plugin was catalogued" >&2; fail=1; }
jq -e 'any(.plugins[]; .name == "api-go")' "$work/new.json" >/dev/null || { echo "FAIL: a released plugin vanished" >&2; fail=1; }

# 8. A released plugin whose tag is missing (a tagless clone) must error, not be omitted.
fresh
mkdir -p "$work/notags"
git -C "$work/notags" init -q
CATALOGUE_TAG_REPO="$work/notags" expect_fail "build-catalogue.sh with a released plugin's tag missing" \
  "$work/r/scripts/build-catalogue.sh" --out "$work/cat.json"

# 9. A template citing an internal reference is refused; colours and in-string ids are not one.
fresh
mkdir -p "$work/r/plugins/auth/templates"
printf 'a { color: #000; border: 1px solid #333; background: #767676 }\nid = "inv#009"\n' \
  >"$work/r/plugins/auth/templates/ok.css"
"$work/r/scripts/validate-plugins.sh" >/dev/null || { echo "FAIL: a colour or an in-string id read as an issue" >&2; fail=1; }
for ref in '# see #3022' '# ADR-0085' '# albitor#3178' '# albitor-ltd/albitor issue'; do
  printf '%s\n' "$ref" >"$work/r/plugins/auth/templates/ref.sh"
  expect_fail "validate-plugins.sh with a template citing '$ref'" "$work/r/scripts/validate-plugins.sh"
done

exit "$fail"
