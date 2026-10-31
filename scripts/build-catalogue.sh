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
#          catalogue/legacy-pins.json.
# --check  write nothing; exit 1 if the output differs from the committed catalogue. A catalogued
#          plugin whose plugin.json version is ahead of the catalogued ref and whose new tag does
#          not exist yet is a pending release: its committed entry is kept (its own tag and sha
#          must still resolve) and a note names the pending tag.
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
    -h | --help) sed -n '2,18p' "${BASH_SOURCE[0]}"; exit 0 ;;
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
    # A plugin absent from the committed catalogue has never shipped: until it is tagged and the
    # catalogue regenerated it stays out, so a new plugin's pull request can pass --check. A plugin
    # already in the committed catalogue must have its tag; a missing one is an error, never a
    # silent omission.
    if ! sha="$(git -C "${CATALOGUE_TAG_REPO:-$root}" rev-parse --verify --quiet "refs/tags/$name/v$version^{commit}")"; then
      if jq -e --arg n "$name" 'any(.plugins[]?; .name == $n)' "$root/.claude-plugin/marketplace.json" >/dev/null 2>&1; then
        # Pending release (--check only): the version is ahead of the catalogued ref and its tag
        # is not cut yet. Keep the committed entry, provided its own tag and sha still resolve.
        # Generation never takes this path: it must not pin a tag that does not exist.
        if [[ $check -eq 1 ]]; then
          cat_ref="$(jq -r --arg n "$name" '.plugins[] | select(.name == $n) | .source.ref // ""' "$root/.claude-plugin/marketplace.json" | head -n1)"
          cat_sha="$(jq -r --arg n "$name" '.plugins[] | select(.name == $n) | .source.sha // ""' "$root/.claude-plugin/marketplace.json" | head -n1)"
          cat_version="${cat_ref#"$name"/v}"
          semver='^[0-9]+\.[0-9]+\.[0-9]+$'
          if [[ "$cat_ref" == "$name/v"* && "$version" =~ $semver && "$cat_version" =~ $semver &&
            "$cat_version" != "$version" &&
            "$(printf '%s\n%s\n' "$cat_version" "$version" | sort -V | tail -n1)" == "$version" ]]; then
            cat_actual="$(git -C "${CATALOGUE_TAG_REPO:-$root}" rev-parse --verify --quiet "refs/tags/$cat_ref^{commit}" || true)"
            if [[ -z "$cat_actual" || "$cat_actual" != "$cat_sha" ]]; then
              echo "error: $name is pending $name/v$version but its catalogued tag $cat_ref does not resolve to $cat_sha" >&2
              exit 1
            fi
            echo "note: $name is a pending release: $name/v$version is not tagged yet, catalogue keeps $cat_ref" >&2
            versions="$(jq --arg n "$name" --arg v "$cat_version" '. + {($n): $v}' <<<"$versions")"
            shas="$(jq --arg n "$name" --arg s "$cat_sha" '. + {($n): $s}' <<<"$shas")"
            continue
          fi
        fi
        echo "error: tag $name/v$version does not exist (tag the release before regenerating)" >&2
        exit 1
      fi
      echo "note: $name is new and has no release tag yet, left out of the catalogue" >&2
      continue
    fi
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
    --slurpfile pins "$root/catalogue/legacy-pins.json" '
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
          | select(if $mode == "monorepo" then $versions[.key] != null else $pins[0][.key] != null end)
          | .key as $n | {name: $n, source: source($n)} + .value]
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
