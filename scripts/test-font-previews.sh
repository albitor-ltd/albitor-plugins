#!/usr/bin/env bash
# Prove the font preview drift test passes on honest previews and fails on each kind of drift.
# Works on a copy of two small plugins (one with a Reserved Font Name) and a scratch albitor tree;
# nothing outside the temp dir changes. Not part of `just ci` (it is slow and needs fontTools).
# The mutation functions are called indirectly by case_.
# shellcheck disable=SC2329
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fail=0
plugs=(font-geometric font-technical)

fresh() { # a clean copy of the plugins and previews
  rm -rf "$tmp/plugins" "$tmp/albitor"
  mkdir -p "$tmp/plugins" "$tmp/albitor/web/static/fonts"
  for p in "${plugs[@]}"; do cp -r "$root/plugins/$p" "$tmp/plugins/$p"; done
  PLUGINS_DIR="$tmp/plugins" "$root/scripts/subset-previews.sh" "$tmp/albitor" "${plugs[@]}" 2>/dev/null
}
run() { PLUGINS_DIR="$tmp/plugins" "$root/scripts/check-font-previews.sh" "$tmp/albitor" "${plugs[@]}" 2>&1; }

case_() { # case_ accept|refuse <name> <mutation> [substring a refusal must carry]
  local want="$1" name="$2" mut="$3" msg="${4:-}" out got
  fresh; (cd "$tmp" && "$mut")
  if out="$(run)"; then got=accept; else got=refuse; fi
  [[ "$got" == "$want" ]] || { echo "FAIL: $name: wanted $want, got $got: $out" >&2; fail=1; return; }
  if [[ -n "$msg" && "$out" != *"$msg"* ]]; then echo "FAIL: $name: refused without '$msg': $out" >&2; fail=1; fi
}

noop() { :; }
case_ accept "honest previews" noop
tamper_plugin_file() { printf 'x' >>"$tmp/plugins/font-geometric/templates/web/static/fonts/plus-jakarta-sans/PlusJakartaSans-Variable.woff2"; }
case_ refuse "tampered plugin font file" tamper_plugin_file "does not match its recorded sha256"
retag() { jq '.fonts[0].sha256 = "0000000000000000000000000000000000000000000000000000000000000000"' "$tmp/plugins/font-geometric/.claude-plugin/plugin.json" >"$tmp/m" && mv "$tmp/m" "$tmp/plugins/font-geometric/.claude-plugin/plugin.json"; }
case_ refuse "plugin hash re-recorded without regenerating" retag "full sha256 in manifest"
bump() { jq '.version = "0.2.0"' "$tmp/plugins/font-geometric/.claude-plugin/plugin.json" >"$tmp/m" && mv "$tmp/m" "$tmp/plugins/font-geometric/.claude-plugin/plugin.json"; }
case_ refuse "plugin version bumped" bump "plugin_version"
missing() { rm "$tmp"/albitor/web/static/fonts/looks/font-geometric/*/*.woff2; }
case_ refuse "preview file missing" missing "is missing"
corrupt() { printf 'x' >>"$tmp"/albitor/web/static/fonts/looks/font-geometric/*/*.woff2; }
case_ refuse "preview bytes changed" corrupt "differs from its manifest"
stale() { jq '.files[0].preview_sha256 = "00"' "$tmp"/albitor/web/static/fonts/looks/font-geometric/*/manifest.json >"$tmp/m" && mv "$tmp/m" "$(ls "$tmp"/albitor/web/static/fonts/looks/font-geometric/*/manifest.json)"; }
case_ refuse "manifest hash stale" stale "differs from its manifest"
extra() { cp "$tmp"/albitor/web/static/fonts/looks/font-geometric/*/*.woff2 "$tmp/albitor/web/static/fonts/looks/font-geometric/extra.woff2"; }
case_ refuse "unlisted extra preview" extra "do not reproduce"
nolic() { rm "$tmp"/albitor/web/static/fonts/looks/font-geometric/*/OFL.txt; }
case_ refuse "licence text missing" nolic "OFL.txt is missing"
editlic() { echo "extra term" >>"$tmp"/albitor/web/static/fonts/looks/font-geometric/*/OFL.txt; }
case_ refuse "licence text altered" editlic "OFL.txt differs"
# a renamed family whose preview directory and file keep the RFN in the path (bytes and hashes otherwise honest)
rfnpath() {
  local l="$tmp/albitor/web/static/fonts/looks/font-technical" d f
  d="$l/preview-sans"; f="$d/preview-sans-400-normal.woff2"
  mv "$f" "$d/plex-400-normal.woff2"
  jq '(.files[]|select(.preview=="preview-sans-400-normal.woff2")|.preview) = "plex-400-normal.woff2"' "$d/manifest.json" >"$tmp/m" && mv "$tmp/m" "$d/manifest.json"
}
case_ refuse "preview path keeps its Reserved Font Name" rfnpath "preview path keeps Reserved Font Name"
# a preview that is the plugin's own file, unrenamed, standing in for a Reserved Font Name family
unrenamed() {
  local d f; d="$(ls -d "$tmp"/albitor/web/static/fonts/looks/font-technical/preview-sans)"; f="$d/preview-sans-400-normal.woff2"
  # re-cut without the rename: copy the unmodified source, then fix the manifest hash so only the RFN test can object
  cp "$tmp/plugins/font-technical/templates/web/static/fonts/ibm-plex-sans/IBMPlexSans-Regular.woff2" "$f"
  local s; s="$(sha256sum "$f" | cut -d' ' -f1)"
  jq --arg s "$s" '(.files[]|select(.preview=="preview-sans-400-normal.woff2")|.preview_sha256) = $s' "$d/manifest.json" >"$tmp/m" && mv "$tmp/m" "$d/manifest.json"
}
case_ refuse "preview keeps its Reserved Font Name" unrenamed "keeps Reserved Font Name"

[[ $fail -eq 0 ]] && echo "ok: font preview drift test accepts honest previews and refuses each drift"
exit $fail
