#!/usr/bin/env bash
# Validate every plugins/*/.claude-plugin/plugin.json and its catalogue metadata:
# the JSON parses, name matches the directory, version is semver, and catalogue/metadata.json
# lists exactly the plugins that exist, with the plugin.json description.
#
#   validate-plugins.sh [--mode monorepo|legacy]
#
# Only legacy mode (the old per-plugin repositories) requires a plugin to have a pin in
# catalogue/legacy-pins.json; in monorepo mode (the default) the pins are history.
set -euo pipefail

mode=monorepo
case "${1:-}" in
  "") ;;
  --mode) mode="${2:?--mode needs a value}" ;;
  *) echo "usage: $0 [--mode monorepo|legacy]" >&2; exit 2 ;;
esac
[[ "$mode" == monorepo || "$mode" == legacy ]] || { echo "mode must be legacy or monorepo" >&2; exit 2; }

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
  [[ "$(jq -r --arg n "$base" '.[$n].description // ""' "$root/catalogue/metadata.json")" == "$(jq -r '.description // ""' "$manifest")" ]] ||
    err "$base: catalogue/metadata.json description differs from plugin.json"
  # A schema-2 plugin was born in this repository: it has no old repository to pin in legacy mode.
  # A legacy-pinned plugin is released from its old repo and must stay schema 1; flipping it to 2
  # would drop it from the legacy catalogue without a word.
  if [[ "$(jq -r '.schema // 1' "$manifest")" == 2 ]]; then
    ! jq -e --arg n "$base" 'has($n)' "$root/catalogue/legacy-pins.json" >/dev/null ||
      err "$base: schema 2 but listed in catalogue/legacy-pins.json (a legacy plugin must stay schema 1)"
  elif [[ "$mode" == legacy ]]; then
    jq -e --arg n "$base" 'has($n)' "$root/catalogue/legacy-pins.json" >/dev/null ||
      err "$base: no entry in catalogue/legacy-pins.json"
  fi
done

for f in metadata legacy-pins; do
  while IFS= read -r key; do
    [[ -d "$root/plugins/$key" ]] || err "catalogue/$f.json lists '$key' but plugins/$key does not exist"
  done < <(jq -r 'keys_unsorted[]' "$root/catalogue/$f.json")
done

# A kit's tokens.css is read by text (certification and the kit's own contrast test take the first
# root rule), so its header, everything before the `albitor-kit` layer, must not mention the root
# selector or carry a brace.
while IFS= read -r css; do
  header="$(sed '/@layer albitor-kit/q' "$css" | sed '$d')"
  if grep -q -e ':root' -e '[{}]' <<<"$header"; then
    err "${css#"$root"/}: header before '@layer albitor-kit' must contain neither ':root' nor a brace"
  fi
done < <(grep -rl --include=tokens.css '@layer albitor-kit' "$root/plugins" || true)

# A template is rendered into the customer's repository, whose readers cannot resolve the platform's
# issue numbers, ADR ids or its core repository: keep the reason, drop the reference. A hash counts
# as an issue only after a space, a bracket or a comma, as four or five digits with no leading zero,
# and with no hex digit or ';' after them, so a hex colour (#000, #333, #767676, #ffffff80, #1234;)
# or an id inside a string ("inv#009") is not one. The short form albitor#NNNN always is.
templates=("$root"/plugins/*/templates)
if [[ ${#templates[@]} -gt 0 ]]; then
  while IFS= read -r hit; do
    err "${hit#"$root"/}: a template cites an internal reference (issue, ADR or core repository)"
  done < <(grep -rnEI 'ADR-[0-9]{4}|albitor-ltd/albitor([^-]|$)|albitor#[0-9]+|(^|[[:space:](,])#[1-9][0-9]{3,4}([^0-9A-Fa-f;]|$)' \
    "${templates[@]}" 2>/dev/null || true)
fi

if [[ $fail -ne 0 ]]; then
  exit 1
fi
echo "validated ${#dirs[@]} plugins"
