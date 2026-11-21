#!/usr/bin/env bash
# Heading steps must read as steps. Every look (look.json) and the kit-shadcn defaults
# (preview-tokens.json) resolve h1 > h2 > h3 > h4 to the title, heading, lead and body sizes, where
# the platform derives lead as the midpoint of body and heading (app/core/look_css.go). Adjacent
# levels must differ by at least 1.15x. Sizes are in rem. Run from the repo root.
set -euo pipefail
cd "$(dirname "$0")/.."

min=1.15
fail=0
files=(plugins/kit-shadcn/preview-tokens.json plugins/look-*/templates/web/src/lib/look/look.json)
for f in "${files[@]}"; do
  if ! out=$(jq -r --argjson min "$min" '
    def rem: if type == "string" and endswith("rem") then sub("rem$"; "") | tonumber
             else error("type_scale value \(.) is not in rem") end;
    .type_scale as $t
    | ($t.body | rem) as $b | ($t.heading | rem) as $h | ($t.title | rem) as $T
    | (($b + $h) / 2) as $l
    | [["h1/h2", $T, $h], ["h2/h3", $h, $l], ["h3/h4", $l, $b]]
    | map(select(.[1] / .[2] < $min) | "\(.[0]) is \(.[1] / .[2] * 100 | round / 100)x")
    | .[]' "$f"); then
    echo "check-type-scale: $f: cannot read the type scale" >&2
    fail=1
    continue
  fi
  if [ -n "$out" ]; then
    while IFS= read -r line; do echo "check-type-scale: $f: $line, want at least ${min}x" >&2; done <<<"$out"
    fail=1
  fi
done
[ "$fail" -eq 0 ] && echo "check-type-scale: ${#files[@]} type scales step by at least ${min}x"
exit "$fail"
