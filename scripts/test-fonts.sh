#!/usr/bin/env bash
# Prove the font self-hosting gate accepts a good typeface plugin and refuses each kind of bad one.
# Each case copies tests/fonts/valid, applies a mutation (a shell function) and checks the verdict.
# The mutation functions are called indirectly by case_.
# shellcheck disable=SC2329
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
validator="$root/scripts/validate-fonts.sh"
base="$root/tests/fonts/valid"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fail=0
manifest=.claude-plugin/plugin.json
font=fonts/fixture-sans/FixtureSans.woff2
css=fonts/fonts.css
ofl=fonts/fixture-sans/OFL.txt

case_() { # case_ accept|refuse <name> <mutation function> [error substring a refusal must carry]
  local want="$1" name="$2" mut="$3" msg="${4:-}" d="$tmp/case" got out
  rm -rf "$d"; cp -r "$base" "$d"
  (cd "$d" && "$mut")
  if out="$("$validator" "$d" 2>&1)"; then got=accept; else got=refuse; fi
  [[ "$got" == "$want" ]] || { echo "FAIL: $name: wanted $want, got $got: $out" >&2; fail=1; return; }
  if [[ -n "$msg" && "$out" != *"$msg"* ]]; then
    echo "FAIL: $name: refused without '$msg': $out" >&2; fail=1
  fi
}

jqm() { local t; t="$(mktemp)"; jq "$1" "$manifest" >"$t"; mv "$t" "$manifest"; }
resha() { # re-record the font's sha256 after a mutation of the file
  local s; s="$(sha256sum "$font" | cut -d' ' -f1)"; jqm ".fonts[0].sha256 = \"$s\""
}

"$validator" "$base" >/dev/null || { echo "FAIL: tests/fonts/valid refused" >&2; fail=1; }

noop() { :; }
case_ accept "no-op" noop
rfn_none() { jqm '.fonts[0].rfn = "none"'; sed -i "s/, with Reserved Font Name 'Fixture'//" "$ofl"; }
case_ accept "rfn none with no declaration" rfn_none
variable() { jqm '.fonts[0].weight = "200 900"'; }
case_ accept "variable weight range" variable
data_small() { echo '@font-face{font-family:"X";font-display:swap;src:url(data:font/woff2;base64,AAAA) format("woff2")}' >>"$css"; }
case_ accept "small data: URL" data_small
root_rel() { sed -i 's#url("fixture-sans/FixtureSans.woff2")#url("/fonts/fixture-sans/FixtureSans.woff2")#' "$css"; }
case_ accept "root-relative src" root_rel
fallback_face() { printf '@font-face{font-family:"X Fallback";src:local("Arial");size-adjust:97%%}\n' >>"$css"; }
case_ accept "local() fallback face needs no swap" fallback_face
comment_url() { sed -i '1s#.*#/* see https://fonts.googleapis.com for the original */#' "$css"; }
case_ accept "a URL in a comment" comment_url

# Licence
lic_other() { jqm '.fonts[0].licence = "proprietary"'; }
case_ refuse "licence not OFL-1.1" lic_other "is not OFL-1.1"
lic_lower() { jqm '.fonts[0].licence = "OFL"'; }
case_ refuse "licence OFL without version" lic_lower "is not OFL-1.1"
ofl_missing() { rm "$ofl"; }
case_ refuse "OFL.txt missing" ofl_missing "no OFL.txt beside"
ofl_edited() { sed -i 's/PREAMBLE/PREAMBLE (edited)/' "$ofl"; }
case_ refuse "OFL.txt text altered" ofl_edited "canonical OFL-1.1 text"
ofl_other() { printf 'Copyright 2026 Someone\n\nAll rights reserved.\n' >"$ofl"; }
case_ refuse "OFL.txt is another licence" ofl_other "canonical OFL-1.1 text"

# The file
file_missing() { rm "$font"; }
case_ refuse "font file missing" file_missing "does not exist"
sig_bad() { printf 'wOFF-not-woff2' >"$font"; resha; }
case_ refuse "not a woff2 signature" sig_bad "woff2 signature"
ttf() { printf '\0\1\0\0ttf-bytes' >"$font"; resha; }
case_ refuse "ttf bytes" ttf "woff2 signature"
sha_wrong() { printf 'wOF2changed-bytes' >"$font"; }
case_ refuse "sha256 differs" sha_wrong "sha256 differs"
sha_format() { jqm '.fonts[0].sha256 = "ABC"'; }
case_ refuse "sha256 malformed" sha_format "64 lowercase hex"
ext_bad() { cp "$font" fonts/fixture-sans/FixtureSans.ttf; jqm '.fonts[0].file = "fonts/fixture-sans/FixtureSans.ttf"'; }
case_ refuse "file is not .woff2" ext_bad "is not a .woff2 file"
path_escape() { jqm '.fonts[0].file = "../FixtureSans.woff2"'; }
case_ refuse "file escapes the plugin" path_escape "clean relative path"
unlisted() { cp "$font" fonts/fixture-sans/Extra.woff2; }
case_ refuse "font file with no manifest line" unlisted "no fonts entry"

# RFN
rfn_missing() { jqm 'del(.fonts[0].rfn)'; }
case_ refuse "rfn missing" rfn_missing "rfn is required"
rfn_empty() { jqm '.fonts[0].rfn = ""'; }
case_ refuse "rfn empty" rfn_empty "rfn is required"
rfn_none_but_declared() { jqm '.fonts[0].rfn = "none"'; }
case_ refuse "rfn none but the OFL.txt declares one" rfn_none_but_declared "declares a Reserved Font Name"
rfn_wrong() { jqm '.fonts[0].rfn = "Other"'; }
case_ refuse "rfn not in the copyright line" rfn_wrong "not declared in the OFL.txt copyright line"

# Manifest shape
key_missing() { jqm 'del(.fonts[0].source)'; }
case_ refuse "source missing" key_missing "source is required"
key_extra() { jqm '.fonts[0].subset = "latin"'; }
case_ refuse "unknown key" key_extra "unknown key"
no_block() { jqm 'del(.fonts)'; }
case_ refuse "typeface provider with no fonts" no_block "states no fonts block"
empty_block() { jqm '.fonts = []'; }
case_ refuse "empty fonts" empty_block "non-empty array"
not_typeface() { jqm '.provides = ["visual-language@1"]'; }
case_ refuse "fonts on a plugin that is not a typeface" not_typeface "does not provide typeface@1"

# CSS
css_http() { sed -i 's#url("fixture-sans/FixtureSans.woff2")#url("https://cdn.example.com/f.woff2")#' "$css"; }
case_ refuse "absolute https src" css_http "absolute URL"
css_proto() { sed -i 's#url("fixture-sans/FixtureSans.woff2")#url(//cdn.example.com/f.woff2)#' "$css"; }
case_ refuse "protocol-relative src" css_proto "protocol-relative"
css_import() { sed -i '1i @import url("other.css");' "$css"; }
case_ refuse "@import" css_import "@import"
css_google() { sed -i '1i @import url(https://fonts.googleapis.com/css2?family=Nunito);' "$css"; }
case_ refuse "Google Fonts" css_google "third-party font host"
css_gstatic() { sed -i 's#url("fixture-sans/FixtureSans.woff2")#url(https://fonts.gstatic.com/s/x.woff2)#' "$css"; }
case_ refuse "gstatic src" css_gstatic "third-party font host"
css_data() { printf '@font-face{font-family:"X";font-display:swap;src:url(data:font/woff2;base64,%s)}\n' "$(head -c 3000 /dev/zero | tr '\0' 'A')" >>"$css"; }
case_ refuse "large data: URL" css_data "data: URL larger"
css_unresolved() { sed -i 's#fixture-sans/FixtureSans.woff2#fixture-sans/Nope.woff2#' "$css"; }
case_ refuse "src that resolves to no file" css_unresolved "does not resolve"
css_noswap() { sed -i '/font-display/d' "$css"; }
case_ refuse "no font-display swap" css_noswap "without font-display: swap"
css_block() { sed -i 's/swap/block/' "$css"; }
case_ refuse "font-display block" css_block "without font-display: swap"
css_elsewhere() { mkdir -p styles; printf '@font-face{font-family:"X";font-display:swap;src:url(https://x.test/f.woff2)}\n' >styles/extra.css; }
case_ refuse "bad CSS outside fonts.css" css_elsewhere "absolute URL"

[[ $fail -eq 0 ]] && echo "font gate tests passed"
exit "$fail"
