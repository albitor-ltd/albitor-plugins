#!/usr/bin/env bash
# Prove check-release-paths.sh accepts a release-safe template path and refuses each bad one,
# naming the file.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
check="$root/scripts/check-release-paths.sh"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fail=0

case_() { # case_ accept|refuse <template-relative path>
  local want="$1" rel="$2" d="$tmp/plugins" got out
  rm -rf "$d"; mkdir -p "$d/p/templates/web/$(dirname "$rel")"
  : >"$d/p/templates/web/$rel"
  if out="$("$check" "$d" 2>&1)"; then got=accept; else got=refuse; fi
  if [[ "$got" != "$want" ]]; then echo "FAIL: '$rel': wanted $want, got $got: $out" >&2; fail=1; return; fi
  if [[ "$want" == refuse && "$out" != *"web/$rel"* ]]; then echo "FAIL: '$rel': refusal does not name the file: $out" >&2; fail=1; fi
}

case_ accept "static/fonts/space-grotesk/SpaceGrotesk-wght.woff2"
case_ accept "static/a_b~c@d+e=f,g.h-i.txt"
case_ refuse "static/fonts/SpaceGrotesk[wght].woff2"
case_ refuse "static/a b.html"
case_ refuse "static/caf$(printf '\303\251').png"
case_ refuse "static/logo(1).png"
case_ refuse "$(printf 'static/a\nb.png')"
# 255-byte name limit: build the over-long path from nested directories instead
deep="$tmp/long/p/templates/web"; for _ in 1 2 3 4 5 6; do deep="$deep/$(printf 'a%.0s' $(seq 1 100))"; done
mkdir -p "$deep"; : >"$deep/f.txt"
if "$check" "$tmp/long" >/dev/null 2>&1; then echo "FAIL: a path over 505 characters was accepted" >&2; fail=1; fi
"$check" "$root/plugins" >/dev/null || { echo "FAIL: the real plugins are refused" >&2; fail=1; }
exit "$fail"
