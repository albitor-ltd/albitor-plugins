#!/usr/bin/env bash
# Generate .claude-plugin/marketplace.json from the plugins/ tree.
#
#   build-catalogue.sh [--mode legacy|monorepo] [--check] [--out FILE]
#
# monorepo (default) each plugin points at this repo as a git-subdir source pinned to the tag
#          <name>/v<version>, <version> being read from plugins/<name>/.claude-plugin/plugin.json,
#          and to the commit that tag resolves to (sha), so a moved tag fails --check. Needs the
#          tags locally; CATALOGUE_TAG_REPO names another checkout to resolve them in.
# legacy   each plugin points at its old albitor-plugin-<name> repo and ref from
#          catalogue/legacy-pins.json, and the retired visual-* entries are carried through.
# --check  write nothing; exit 1 if the output differs from the committed catalogue.
#
# Catalogue-facing metadata (description, category, keywords, license) lives in
# catalogue/metadata.json; the key order there is the catalogue order. A plugin with no legacy pin
# (a schema-2 plugin born in this repository) appears only in monorepo mode.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mode=monorepo
check=0
out="$root/.claude-plugin/marketplace.json"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode) mode="${2:?--mode needs a value}"; shift 2 ;;
    --check) check=1; shift ;;
    --out) out="${2:?--out needs a value}"; shift 2 ;;
    -h | --help) sed -n '2,15p' "${BASH_SOURCE[0]}"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
if [[ "$mode" != legacy && "$mode" != monorepo ]]; then
  echo "mode must be legacy or monorepo" >&2
  exit 2
fi

command -v jq >/dev/null || { echo "jq is required" >&2; exit 2; }

versions='{}'
shas='{}'
if [[ "$mode" == monorepo ]]; then
  while IFS= read -r name; do
    manifest="$root/plugins/$name/.claude-plugin/plugin.json"
    [[ -f "$manifest" ]] || { echo "missing $manifest" >&2; exit 1; }
    version="$(jq -r '.version // ""' "$manifest")"
    if [[ -z "$version" || "$version" == null ]]; then
      echo "error: $name has no version in $manifest" >&2
      exit 1
    fi
    # A plugin whose metadata says "unreleased": true has no tag yet and is left out of the
    # monorepo catalogue. The flag is only for that gap: once the tag exists it must come out in
    # the PR that regenerates the catalogue, so it cannot hide a released plugin.
    if [[ "$(jq -r --arg n "$name" '.[$n].unreleased // false' "$root/catalogue/metadata.json")" == true ]]; then
      if git -C "${CATALOGUE_TAG_REPO:-$root}" rev-parse --verify --quiet "refs/tags/$name/v$version^{commit}" >/dev/null; then
        echo "error: $name/v$version is tagged but catalogue/metadata.json still says unreleased" >&2
        exit 1
      fi
      continue
    fi
    sha="$(git -C "${CATALOGUE_TAG_REPO:-$root}" rev-parse --verify --quiet "refs/tags/$name/v$version^{commit}")" || {
      echo "error: tag $name/v$version does not exist (tag the release before regenerating)" >&2
      exit 1
    }
    versions="$(jq --arg n "$name" --arg v "$version" '. + {($n): $v}' <<<"$versions")"
    shas="$(jq --arg n "$name" --arg s "$sha" '. + {($n): $s}' <<<"$shas")"
  done < <(jq -r 'keys_unsorted[]' "$root/catalogue/metadata.json")
fi

# Legacy mode points every plugin at its old repo: a schema-1 plugin with no pin is an error,
# never a silent omission. Only a schema-2 plugin (born here, no old repo) may lack one.
if [[ "$mode" == legacy ]]; then
  while IFS= read -r name; do
    manifest="$root/plugins/$name/.claude-plugin/plugin.json"
    [[ -f "$manifest" ]] || { echo "missing $manifest" >&2; exit 1; }
    [[ "$(jq -r '.schema // 1' "$manifest")" == 2 ]] && continue
    if ! jq -e --arg n "$name" 'has($n)' "$root/catalogue/legacy-pins.json" >/dev/null; then
      echo "error: $name is a schema-1 plugin with no entry in catalogue/legacy-pins.json" >&2
      exit 1
    fi
  done < <(jq -r 'keys_unsorted[]' "$root/catalogue/metadata.json")
fi

render() {
  jq -a -n \
    --arg mode "$mode" \
    --arg desc "Albitor's catalogue of Claude Code plugins." \
    --argjson versions "$versions" \
    --argjson shas "$shas" \
    --slurpfile meta "$root/catalogue/metadata.json" \
    --slurpfile pins "$root/catalogue/legacy-pins.json" \
    --slurpfile retired "$root/catalogue/retired-entries.json" '
    def source($n):
      if $mode == "monorepo" then
        {source: "git-subdir", url: "albitor-ltd/albitor-plugins",
         path: ("plugins/" + $n), ref: ($n + "/v" + $versions[$n]), sha: $shas[$n]}
      else
        {source: "github", repo: $pins[0][$n].repo, ref: $pins[0][$n].ref}
      end;
    {
      "$schema": "https://anthropic.com/claude-code/marketplace.schema.json",
      name: "albitor-plugins",
      description: $desc,
      owner: {name: "Albitor Ltd", email: "jamie@albitor.com"},
      plugins: (
        [$meta[0] | to_entries[]
          | select(.value.unreleased != true)
          | select($mode == "monorepo" or $pins[0][.key] != null)
          | .key as $n | {name: $n, source: source($n)} + .value]
        + (if $mode == "legacy" then $retired[0] else [] end)
      )
    }'
}

if [[ $check -eq 1 ]]; then
  if ! diff -u "$out" <(render); then
    echo "catalogue drift: run scripts/build-catalogue.sh --mode $mode" >&2
    exit 1
  fi
  echo "catalogue up to date ($mode)"
else
  render >"$out"
  echo "wrote $out ($mode)"
fi
