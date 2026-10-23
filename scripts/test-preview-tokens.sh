#!/usr/bin/env bash
# Prove the preview-tokens validator accepts a good file and refuses each kind of bad one.
# Each case mutates tests/preview-tokens/valid.json with a jq expression and must be refused.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
validator="$root/scripts/validate-preview-tokens.sh"
base="$root/tests/preview-tokens/valid.json"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fail=0

expect() { # expect accept|refuse <name> <jq mutation> [error substring a refusal must carry]
  local want="$1" name="$2" mut="$3" msg="${4:-}" f="$tmp/case.json"
  jq "$mut" "$base" >"$f"
  check "$want" "$name" "$f" "$msg"
}

check() { # check accept|refuse <name> <file> [error substring]
  local want="$1" name="$2" f="$3" msg="${4:-}" got out
  if out="$("$validator" "$f" 2>&1)"; then got=accept; else got=refuse; fi
  [[ "$got" == "$want" ]] || { echo "FAIL: $name: wanted $want, got $got" >&2; fail=1; return; }
  if [[ -n "$msg" && "$out" != *"$msg"* ]]; then
    echo "FAIL: $name: refused without '$msg': $out" >&2; fail=1
  fi
}

"$validator" "$base" >/dev/null || { echo "FAIL: valid.json refused" >&2; fail=1; }
expect accept "no-op" '.'
expect accept "all optionals" '. + {heading_font_family:"Georgia, serif",border_width:"2px",min_target:"24px",heading_tracking:"-0.02em",focus_ring:"3px",tabular_figures:true,dark_target:7} | .elevation="layered" | .colours += {control_border:"#777",danger:"#d4351c",success:"#00703c",warning:"#ffdd00",warning_alt:"#f47738"}'
expect accept "roles" '. + {roles:{header:{background:"#161616",foreground:"#fff"},header_withdrawn:{background:"#1d70b8",foreground:"#ffffff"},link:"#0f62fe",focus:"#ffdd00",focus_ink:"#0b0c0c"}}'
expect refuse "roles bad focus_ink" '. + {roles:{focus_ink:"black"}}' "roles.focus_ink is not a #rgb"
expect accept "roles link only" '. + {roles:{link:"#0f62fe"}}'
expect accept "roles null" '. + {roles:null}'
expect refuse "roles not an object" '. + {roles:"x"}' "roles must be an object"
expect refuse "roles unknown key" '. + {roles:{sidebar:"#fff"}}' "roles.sidebar: unknown field"
expect refuse "roles bad link" '. + {roles:{link:"blue"}}' "roles.link is not a #rgb"
expect refuse "roles css injection" '. + {roles:{focus:"#fff;}"}}' "roles.focus is not a #rgb"
expect refuse "roles header without foreground" '. + {roles:{header:{background:"#000"}}}' "roles.header.foreground is required"
expect refuse "roles withdrawn bad bg" '. + {roles:{header_withdrawn:{background:"url(x)",foreground:"#fff"}}}' "roles.header_withdrawn.background is not a #rgb"
expect refuse "roles header unknown member" '. + {roles:{header:{background:"#000",foreground:"#fff",x:"#fff"}}}' "roles.header.x: unknown field"
expect accept "tracking zero" '.heading_tracking="0"'
expect accept "4 decimals" '.radius="0.1234rem"'
expect accept "dark_target zero" '.dark_target=0'
expect accept "empty optional" '.border_width=""'

expect refuse "unknown top-level key" '.surprise=1' "surprise: unknown field"
expect refuse "unknown colours key" '.colours.hotpink="#f0f"' "colours.hotpink: unknown field"
expect refuse "unknown type_scale key" '.type_scale.huge="48px"' "type_scale.huge: unknown field"
expect refuse "missing version" 'del(.version)' "version is required"
expect refuse "wrong version" '.version=2' "version is not 1"
expect refuse "missing required colour" 'del(.colours.accent)' "colours.accent is required"
expect refuse "missing font_family" 'del(.font_family)' "font_family is required"
expect refuse "missing type_scale entry" 'del(.type_scale.title)' "type_scale.title is required"
expect refuse "5 decimals" '.radius="0.12345rem"' "radius is not a px or rem length"
expect refuse "unit em for length" '.radius="1em"' "radius is not a px or rem length"
expect refuse "bare number length" '.radius=4' "radius must be a string"
expect refuse "4-digit length" '.radius="1000px"' "radius is not a px or rem length"
expect refuse "heading_tracking px" '.heading_tracking="2px"' "heading_tracking is not 0 or an em value"
expect refuse "heading_tracking >= 1em" '.heading_tracking="1.5em"' "heading_tracking is not 0 or an em value"
expect refuse "heading_tracking 4 decimals" '.heading_tracking="-0.0123em"' "heading_tracking is not 0 or an em value"
expect refuse "bad hex" '.colours.text="red"' "colours.text is not a #rgb"
expect refuse "hex 4 digits" '.colours.text="#abcd"' "colours.text is not a #rgb"
expect refuse "css injection in colour" '.colours.text="#000; background: red"' "colours.text is not a #rgb"
expect refuse "bad optional colour" '.colours.danger="rgb(0,0,0)"' "colours.danger is not a #rgb"
expect refuse "bad density" '.density="roomy"' "density is not one of"
expect refuse "bad elevation" '.elevation="floating"' "elevation is not one of"
expect refuse "heading_weight not hundreds" '.heading_weight=650' "heading_weight is not 100-900"
expect refuse "heading_weight out of range" '.heading_weight=1000' "heading_weight is not 100-900"
expect refuse "dark_target too low" '.dark_target=3' "dark_target is not 0 or between"
expect refuse "dark_target too high" '.dark_target=8' "dark_target is not 0 or between"
expect refuse "tabular_figures string" '.tabular_figures="yes"' "tabular_figures must be a boolean"
expect refuse "font injection" '.font_family="arial; color: red"' "font_family has a family name outside"
expect refuse "too many families" '.font_family="a,b,c,d,e,f,g,h,i"' "font_family names more than 8"
expect refuse "string version" '.version="1"' "version is not 1"

expect refuse "dark_target empty string" '.dark_target=""' "dark_target must be a number"
expect refuse "family 241 bytes" '.font_family=("a" * 241)' "font_family is longer than 240"
expect accept "family 240 bytes" '.font_family=(([range(7)|"a" * 28] + ["a" * 30]) | join(", "))'

# Structural cases the mutations above cannot express.
raw() { printf '%b' "$2" >"$tmp/raw.json"; check refuse "$1" "$tmp/raw.json" "$3"; }
raw "array" '[]' "top level is not an object"
raw "invalid JSON" '{' "not valid JSON"
raw "empty file" '' "exactly one JSON value"
jq -c . "$base" >"$tmp/one.json"
{ cat "$tmp/one.json"; cat "$tmp/one.json"; } >"$tmp/two.json"
check refuse "two concatenated values" "$tmp/two.json" "exactly one JSON value"
{ cat "$tmp/one.json"; echo '[]'; } >"$tmp/objarr.json"
check refuse "object then array" "$tmp/objarr.json" "exactly one JSON value"
for lit in '1.0' '1e0'; do
  jq -c . "$base" | sed "s/\"version\":1/\"version\":$lit/" >"$tmp/ver.json"
  check refuse "version $lit" "$tmp/ver.json" "integer literals"
done
for lit in '700.0' '7e2'; do
  jq -c . "$base" | sed -E "s/\"heading_weight\":[0-9]+/\"heading_weight\":$lit/" >"$tmp/hw.json"
  check refuse "heading_weight $lit" "$tmp/hw.json" "integer literals"
done
{ printf '\xef\xbb\xbf'; cat "$tmp/one.json"; } >"$tmp/bom.json"
check refuse "UTF-8 BOM" "$tmp/bom.json" "byte order mark"
{ cat "$tmp/one.json"; head -c 4200 /dev/zero | tr '\0' ' '; } >"$tmp/big.json"
check refuse "oversized" "$tmp/big.json" "larger than 4096 bytes"

[[ $fail -eq 0 ]] && echo "preview-tokens validator tests passed"
exit "$fail"
