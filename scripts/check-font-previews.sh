#!/usr/bin/env bash
# The font preview drift test (ADR-0107 section 3): the Look-step previews checked into an albitor
# checkout's web/static/fonts/looks/ must be exactly what subset-previews.sh cuts from this
# repository's typeface plugins today.
#
#   check-font-previews.sh <path to an albitor-ltd/albitor checkout> [plugin...]
#
# Fails on, per plugin font:
#   - a missing manifest or preview file, or a preview file no manifest lists;
#   - a manifest whose plugin version, source, rfn or full-file sha256 differs from the plugin's, or
#     whose recorded full sha256 no longer matches the plugin's file on disk;
#   - a preview whose bytes differ from the manifest's recorded sha256;
#   - a regeneration that differs between two runs (not deterministic) or from the checked-in files;
#   - a preview of a family with a Reserved Font Name that still carries it (in a name or a path).
# Like compose-check.sh it is not part of `just ci`: it needs an albitor checkout.
# PLUGINS_DIR overrides the plugins tree (scripts/test-font-previews.sh uses it).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
plugins="${PLUGINS_DIR:-$root/plugins}"
albitor="${1:?usage: check-font-previews.sh <albitor checkout> [plugin...]}"
shift
looks="$albitor/web/static/fonts/looks"
command -v jq >/dev/null || { echo "jq is required" >&2; exit 2; }
# shellcheck disable=SC1091
source "$root/scripts/font-previews-lib.sh"
py="$(font_python)"

names=("$@")
if [[ ${#names[@]} -eq 0 ]]; then
  for m in "$plugins"/*/.claude-plugin/plugin.json; do
    jq -e '(.fonts // []) | length > 0' "$m" >/dev/null && names+=("$(basename "$(dirname "$(dirname "$m")")")")
  done
fi
[[ ${#names[@]} -gt 0 ]] || { echo "no typeface plugins found" >&2; exit 2; }

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fail=0
err() { echo "error: $*" >&2; fail=1; }
sha() { sha256sum "$1" | cut -d' ' -f1; }

# a cut that cannot be made (a plugin file that no longer matches its recorded sha256, a rename
# that keeps its RFN) is itself drift; the per-plugin checks below still say what else is wrong
cut_ok=1
for run in run1 run2; do
  if ! PLUGINS_DIR="$plugins" "$root/scripts/subset-previews.sh" --out "$tmp/$run" "${names[@]}" 2>"$tmp/cut-err"; then
    err "cannot cut previews from the plugins: $(grep -v '^WARNING' "$tmp/cut-err" | tr '\n' ' ')"
    cut_ok=0; break
  fi
done
if [[ $cut_ok -eq 1 ]] && ! diff -rq "$tmp/run1" "$tmp/run2" >/dev/null; then
  err "two runs of subset-previews.sh differ: the cut is not deterministic"
fi

for name in "${names[@]}"; do
  pm="$plugins/$name/.claude-plugin/plugin.json"
  [[ -f "$pm" ]] || { err "$name: no plugin manifest"; continue; }
  version="$(jq -r .version "$pm")"
  [[ -d "$looks/$name" ]] || { err "$name: no previews under $looks/$name"; continue; }

  # every plugin family has a manifest here, agreeing with the plugin
  while IFS= read -r family; do
    mf=""
    for c in "$looks/$name"/*/manifest.json; do
      [[ -f "$c" ]] && [[ "$(jq -r .family "$c")" == "$family" ]] && mf="$c"
    done
    [[ -n "$mf" ]] || { err "$name: no preview manifest for family '$family'"; continue; }
    dir="$(dirname "$mf")"
    [[ "$(jq -r .plugin_version "$mf")" == "$version" ]] || err "$name/$family: manifest plugin_version $(jq -r .plugin_version "$mf") != plugin $version"
    [[ "$(jq -r .plugin "$mf")" == "$name" ]] || err "$name/$family: manifest names plugin $(jq -r .plugin "$mf")"
    rfn="$(jq -r --arg f "$family" '[.fonts[]|select(.family==$f)][0].rfn' "$pm")"
    [[ "$(jq -r .rfn "$mf")" == "$rfn" ]] || err "$name/$family: manifest rfn '$(jq -r .rfn "$mf")' != plugin rfn '$rfn'"
    [[ "$(jq -r .source "$mf")" == "$(jq -r --arg f "$family" '[.fonts[]|select(.family==$f)][0].source' "$pm")" ]] || err "$name/$family: manifest source differs from the plugin's"

    # each plugin file of the family has a manifest entry whose hashes hold
    while IFS=$'\t' read -r file psha; do
      msha="$(jq -r --arg f "$file" '[.files[]|select(.full_file==$f)][0].full_sha256 // ""' "$mf")"
      [[ -n "$msha" ]] || { err "$name/$family: manifest has no entry for $file"; continue; }
      [[ "$msha" == "$psha" ]] || err "$name/$family: $file full sha256 in manifest ($msha) != plugin's ($psha)"
      if [[ ! -f "$plugins/$name/$file" ]]; then err "$name: plugin file $file is missing"
      elif [[ "$(sha "$plugins/$name/$file")" != "$psha" ]]; then err "$name: plugin file $file does not match its recorded sha256 $psha"; fi
    done < <(jq -r --arg f "$family" '[.fonts[]|select(.family==$f)|[.file,.sha256]]|unique[]|@tsv' "$pm")

    # each listed preview exists with its recorded bytes; none unlisted
    while IFS=$'\t' read -r prev psha; do
      if [[ ! -f "$dir/$prev" ]]; then err "$name/$family: preview file $prev is missing"; continue; fi
      [[ "$(sha "$dir/$prev")" == "$psha" ]] || err "$name/$family: $prev differs from its manifest sha256"
      if [[ "$rfn" != "none" ]]; then
        IFS=',' read -ra rs <<<"${rfn//, /,}"
        "$py" -I "$root/scripts/font-previews.py" rfnscan "$dir/$prev" "${rs[@]}" || err "$name/$family: $prev keeps its Reserved Font Name"
        for r in "${rs[@]}"; do
          [[ "${dir,,}/${prev,,}" != *"${r,,}"* ]] || err "$name/$family: preview path keeps Reserved Font Name '$r'"
        done
      fi
    done < <(jq -r '.files[]|[.preview,.preview_sha256]|@tsv' "$mf")
    while IFS= read -r f; do
      jq -e --arg p "$(basename "$f")" '[.files[].preview]|index($p)' "$mf" >/dev/null || err "$name/$family: $(basename "$f") is not listed in the manifest"
    done < <(find "$dir" -name '*.woff2')
  done < <(jq -r '[.fonts[].family]|unique[]' "$pm")

  # nothing extra, and the checked-in tree is exactly what a fresh cut gives
  if [[ $cut_ok -eq 1 ]] && ! diff -rq "$tmp/run1/$name" "$looks/$name" >"$tmp/diff" 2>&1; then
    err "$name: checked-in previews do not reproduce from the plugin: $(tr '\n' ';' <"$tmp/diff")"
  fi
done

[[ $fail -eq 0 ]] && echo "font previews match the plugins (${#names[@]} plugins)"
exit $fail
