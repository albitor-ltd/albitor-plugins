#!/usr/bin/env bash
# Prove the preview-tokens validator accepts a good file and refuses each kind of bad one.
# Each case mutates tests/preview-tokens/valid.json with a jq expression and must be refused.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
validator="$root/scripts/validate-preview-tokens.sh"
base="$root/tests/preview-tokens/valid.json"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fail=0

expect() { # expect accept|refuse <name> <jq mutation>
  local want="$1" name="$2" mut="$3" f="$tmp/case.json"
  jq "$mut" "$base" >"$f"
  if "$validator" "$f" >/dev/null 2>&1; then got=accept; else got=refuse; fi
  [[ "$got" == "$want" ]] || { echo "FAIL: $name: wanted $want, got $got" >&2; fail=1; }
}

"$validator" "$base" >/dev/null || { echo "FAIL: valid.json refused" >&2; fail=1; }
expect accept "no-op" '.'
expect accept "all optionals" '. + {heading_font_family:"Georgia, serif",border_width:"2px",min_target:"24px",heading_tracking:"-0.02em",focus_ring:"3px",tabular_figures:true,dark_target:7} | .elevation="layered" | .colours += {control_border:"#777",danger:"#d4351c",success:"#00703c",warning:"#ffdd00",warning_alt:"#f47738"}'
expect accept "tracking zero" '.heading_tracking="0"'
expect accept "4 decimals" '.radius="0.1234rem"'
expect accept "dark_target zero" '.dark_target=0'
expect accept "empty optional" '.border_width=""'

expect refuse "unknown top-level key" '.surprise=1'
expect refuse "unknown colours key" '.colours.hotpink="#f0f"'
expect refuse "unknown type_scale key" '.type_scale.huge="48px"'
expect refuse "missing version" 'del(.version)'
expect refuse "wrong version" '.version=2'
expect refuse "missing required colour" 'del(.colours.accent)'
expect refuse "missing font_family" 'del(.font_family)'
expect refuse "missing type_scale entry" 'del(.type_scale.title)'
expect refuse "5 decimals" '.radius="0.12345rem"'
expect refuse "unit em for length" '.radius="1em"'
expect refuse "bare number length" '.radius=4'
expect refuse "4-digit length" '.radius="1000px"'
expect refuse "heading_tracking px" '.heading_tracking="2px"'
expect refuse "heading_tracking >= 1em" '.heading_tracking="1.5em"'
expect refuse "heading_tracking 4 decimals" '.heading_tracking="-0.0123em"'
expect refuse "bad hex" '.colours.text="red"'
expect refuse "hex 4 digits" '.colours.text="#abcd"'
expect refuse "css injection in colour" '.colours.text="#000; background: red"'
expect refuse "bad optional colour" '.colours.danger="rgb(0,0,0)"'
expect refuse "bad density" '.density="roomy"'
expect refuse "bad elevation" '.elevation="floating"'
expect refuse "heading_weight not hundreds" '.heading_weight=650'
expect refuse "heading_weight out of range" '.heading_weight=1000'
expect refuse "dark_target too low" '.dark_target=3'
expect refuse "dark_target too high" '.dark_target=8'
expect refuse "tabular_figures string" '.tabular_figures="yes"'
expect refuse "font injection" '.font_family="arial; color: red"'
expect refuse "too many families" '.font_family="a,b,c,d,e,f,g,h,i"'
expect refuse "string version" '.version="1"'

# Structural cases the mutations above cannot express.
printf '[]' >"$tmp/arr.json"
"$validator" "$tmp/arr.json" >/dev/null 2>&1 && { echo "FAIL: array accepted" >&2; fail=1; }
printf '{' >"$tmp/bad.json"
"$validator" "$tmp/bad.json" >/dev/null 2>&1 && { echo "FAIL: invalid JSON accepted" >&2; fail=1; }
jq '.font_family = ("a" * 5000)' "$base" >"$tmp/big.json" 2>/dev/null || jq '.font_family = ([range(5000)|"a"]|join(""))' "$base" >"$tmp/big.json"
"$validator" "$tmp/big.json" >/dev/null 2>&1 && { echo "FAIL: oversized accepted" >&2; fail=1; }

[[ $fail -eq 0 ]] && echo "preview-tokens validator tests passed"
exit "$fail"
