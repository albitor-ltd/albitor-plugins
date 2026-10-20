#!/usr/bin/env bash
# Validate every plugins/*/preview-tokens.json against the contract the platform reads
# (governance.PreviewTokens in albitor-ltd/albitor, app/core/governance/preview_tokens.go).
# The platform parses with DisallowUnknownFields and ignores a file that fails, so a typo here
# would silently drop a pack's look: unknown keys, bad lengths, colours and enums are refused.
#
#   validate-preview-tokens.sh [file...]   (default: plugins/*/preview-tokens.json)
#
# Keep in step with the Go contract; tests/preview-tokens/ holds the accepted and refused fixtures.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command -v jq >/dev/null || { echo "jq is required" >&2; exit 2; }

read -r -d '' filter <<'JQ' || true
def isobj: type == "object";
def hex: "\\A#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})\\z";
def len: "\\A(?:0|[0-9]{1,3}(?:\\.[0-9]{1,4})?(?:px|rem))\\z";
def qfam: "\\A\"[A-Za-z0-9][A-Za-z0-9 \\-]{0,47}\"\\z";
def bfam: "\\A[A-Za-z][A-Za-z0-9\\-]{0,31}(?: [A-Za-z0-9][A-Za-z0-9\\-]{0,31}){0,3}\\z";
# Optional fields: absent, null or "" read as unset on the platform side.
def unset: . == null or . == "";
def lenerr: if type != "string" then "must be a string" elif test(len) then empty else "is not a px or rem length" end;
def hexerr: if type != "string" then "must be a string" elif test(hex) then empty else "is not a #rgb or #rrggbb colour" end;
def fam($required):
  if type != "string" then "must be a string"
  elif . == "" then (if $required then "is required" else empty end)
  elif utf8bytelength > 240 then "is longer than 240 characters"
  else split(",") as $p
    | if ($p|length) > 8 then "names more than 8 families"
      elif ([$p[] | gsub("^\\s+|\\s+$"; "") | select((test(qfam) or test(bfam)) | not)] | length) > 0
        then "has a family name outside the allowed characters"
      else empty end
  end;
def req(path; f): (try getpath(path) catch null) as $v
  | if $v == null then "\(path | join(".")) is required" else ($v | f | "\(path | join(".")) \(.)") end;
def opt(path; f): (try getpath(path) catch null) as $v
  | if ($v | unset) then empty else ($v | f | "\(path | join(".")) \(.)") end;
def unknown(allowed; prefix): keys_unsorted[] | select(. as $k | allowed | index($k) | not) | "\(prefix)\(.): unknown field";
def enum(vals): if type != "string" then "must be a string" elif IN(vals[]) then empty else "is not one of \(vals | join(", "))" end;

if (isobj | not) then "top level is not an object" else
  . as $d
  | ( unknown(["version","font_family","heading_font_family","heading_weight","type_scale","radius","density","elevation","border_width","min_target","heading_tracking","focus_ring","tabular_figures","dark_target","colours"]; ""),
      (if ($d.type_scale|isobj) then ($d.type_scale | unknown(["small","body","heading","title"]; "type_scale.")) else empty end),
      (if ($d.colours|isobj) then ($d.colours | unknown(["text","text_secondary","border","surface","background","accent","on_accent","control_border","danger","success","warning","warning_alt"]; "colours.")) else empty end),
      req(["version"]; if . == 1 then empty else "is not 1" end),
      req(["font_family"]; fam(true)),
      opt(["heading_font_family"]; fam(false)),
      req(["heading_weight"]; if (type != "number") or (. != floor) then "is not an integer" elif . < 100 or . > 900 or (. % 100) != 0 then "is not 100-900 in hundreds" else empty end),
      (["small","body","heading","title"][] as $k | req(["type_scale",$k]; lenerr)),
      (if ($d.type_scale|isobj) | not then "type_scale is required and must be an object" else empty end),
      req(["radius"]; lenerr),
      req(["density"]; enum(["compact","comfortable","spacious"])),
      req(["elevation"]; enum(["none","subtle","raised","layered"])),
      opt(["border_width"]; lenerr),
      opt(["min_target"]; lenerr),
      opt(["focus_ring"]; lenerr),
      opt(["heading_tracking"]; if type != "string" then "must be a string" elif test("\\A(?:0|-?0\\.[0-9]{1,3}em)\\z") then empty else "is not 0 or an em value such as -0.02em" end),
      (if ($d | has("tabular_figures")) and ($d.tabular_figures != null) and (($d.tabular_figures|type) != "boolean") then "tabular_figures must be a boolean" else empty end),
      opt(["dark_target"]; if type != "number" then "must be a number" elif . == 0 or (. >= 4.5 and . <= 7) then empty else "is not 0 or between 4.5 and 7" end),
      (if ($d.colours|isobj) | not then "colours is required and must be an object" else empty end),
      (["text","text_secondary","border","surface","background","accent","on_accent"][] as $k | req(["colours",$k]; hexerr)),
      (["control_border","danger","success","warning","warning_alt"][] as $k | opt(["colours",$k]; hexerr))
    )
end
JQ

if [[ $# -gt 0 ]]; then
  files=("$@")
else
  shopt -s nullglob
  files=("$root"/plugins/*/preview-tokens.json)
fi
[[ ${#files[@]} -gt 0 ]] || { echo "no preview-tokens.json files found" >&2; exit 2; }

fail=0
for f in "${files[@]}"; do
  rel="${f#"$root"/}"
  if [[ "$(wc -c <"$f")" -gt 4096 ]]; then
    echo "error: $rel: larger than 4096 bytes" >&2; fail=1; continue
  fi
  if ! jq -e . "$f" >/dev/null 2>&1; then
    echo "error: $rel: not valid JSON" >&2; fail=1; continue
  fi
  out="$(jq -r "$filter" "$f")"
  if [[ -n "$out" ]]; then
    while IFS= read -r line; do echo "error: $rel: $line" >&2; done <<<"$out"
    fail=1
  fi
done
[[ $fail -eq 0 ]] && echo "preview-tokens: ${#files[@]} file(s) valid"
exit "$fail"
