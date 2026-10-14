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
  jq -e --arg n "$base" 'has($n)' "$root/catalogue/legacy-pins.json" >/dev/null ||
    err "$base: no entry in catalogue/legacy-pins.json"
done

for f in metadata legacy-pins; do
  while IFS= read -r key; do
    [[ -d "$root/plugins/$key" ]] || err "catalogue/$f.json lists '$key' but plugins/$key does not exist"
  done < <(jq -r 'keys_unsorted[]' "$root/catalogue/$f.json")
done

# Every legacy pin is a repo and ref, or exactly {"monorepoOnly": true}.
while IFS= read -r key; do
  jq -e --arg n "$key" '.[$n] | . == {"monorepoOnly": true} or
      ((.repo | type == "string" and length > 0) and (.ref | type == "string" and length > 0))' \
    "$root/catalogue/legacy-pins.json" >/dev/null 2>&1 ||
    err "legacy-pins.json: '$key' needs repo and ref, or exactly {\"monorepoOnly\": true}"
done < <(jq -r 'keys_unsorted[]' "$root/catalogue/legacy-pins.json")

if [[ $fail -ne 0 ]]; then
  exit 1
fi
echo "validated ${#dirs[@]} plugins"
