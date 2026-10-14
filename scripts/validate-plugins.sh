#!/usr/bin/env bash
# Validate every plugins/*/.claude-plugin/plugin.json and its catalogue metadata:
# the JSON parses, name matches the directory, version is semver, and catalogue/metadata.json
# lists exactly the plugins that exist.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
semver='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'
fail=0
err() { echo "error: $*" >&2; fail=1; }

command -v jq >/dev/null || { echo "jq is required" >&2; exit 2; }

shopt -s nullglob
dirs=("$root"/plugins/*/)
[[ ${#dirs[@]} -gt 0 ]] || err "no plugins found under plugins/"

for dir in "${dirs[@]}"; do
  dir="${dir%/}"
  base="$(basename "$dir")"
  manifest="$dir/.claude-plugin/plugin.json"
  if [[ ! -f "$manifest" ]]; then
    err "$base: missing .claude-plugin/plugin.json"
    continue
  fi
  if ! jq -e . "$manifest" >/dev/null 2>&1; then
    err "$base: plugin.json is not valid JSON"
    continue
  fi
  name="$(jq -r '.name // ""' "$manifest")"
  version="$(jq -r '.version // ""' "$manifest")"
  [[ "$name" == "$base" ]] || err "$base: plugin.json name '$name' does not match directory"
  [[ "$version" =~ $semver ]] || err "$base: version '$version' is not semver"
  [[ -n "$(jq -r '.description // ""' "$manifest")" ]] || err "$base: plugin.json has no description"
  jq -e --arg n "$base" 'has($n)' "$root/catalogue/metadata.json" >/dev/null ||
    err "$base: no entry in catalogue/metadata.json"
  # A schema-2 plugin was born in this repository: it has no old repository to pin in legacy mode.
  # A legacy-pinned plugin is released from its old repo and must stay schema 1; flipping it to 2
  # would drop it from the legacy catalogue without a word.
  if [[ "$(jq -r '.schema // 1' "$manifest")" == 2 ]]; then
    ! jq -e --arg n "$base" 'has($n)' "$root/catalogue/legacy-pins.json" >/dev/null ||
      err "$base: schema 2 but listed in catalogue/legacy-pins.json (a legacy plugin must stay schema 1)"
  else
    jq -e --arg n "$base" 'has($n)' "$root/catalogue/legacy-pins.json" >/dev/null ||
      err "$base: no entry in catalogue/legacy-pins.json"
  fi
done

for f in metadata legacy-pins; do
  while IFS= read -r key; do
    [[ -d "$root/plugins/$key" ]] || err "catalogue/$f.json lists '$key' but plugins/$key does not exist"
  done < <(jq -r 'keys_unsorted[]' "$root/catalogue/$f.json")
done

if [[ $fail -ne 0 ]]; then
  exit 1
fi
echo "validated ${#dirs[@]} plugins"
