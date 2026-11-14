#!/usr/bin/env bash
# Every file a plugin seeds under plugins/*/templates/ must be a path the hosted pipeline's release
# can hold, or the app cannot deploy (albitor-ltd/albitor#4870). The rule, mirrored from
# albitor-hosted-pipeline's release contract: only [A-Za-z0-9._~@+=,/-], at most 512 characters,
# and no empty, "." or ".." segment. The path checked is the one under templates/ (for example
# web/static/fonts/x.woff2). The release path is `static/` plus the path relative to the output
# directory, up to 7 characters longer than the one checked, so the checked limit is 505.
#
#   check-release-paths.sh [plugins-dir]   (default: plugins/ beside this script's repo root)
set -euo pipefail
export LC_ALL=C
allowed='^[A-Za-z0-9._~@+=,/-]+$'
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
plugins="${1:-$root/plugins}"
rule='only [A-Za-z0-9._~@+=,/-], at most 512 characters (505 on the checked path), no empty, "." or ".." segment'
fail=0
while IFS= read -r -d '' f; do
  rel="${f#"$plugins"/}"          # <plugin>/templates/<lane>/...
  p="${rel#*/templates/}"         # <lane>/...
  bad=""
  if [[ ${#p} -gt 505 ]]; then bad="longer than 505 characters (512 less the release's static/ prefix)"
  elif ! [[ "$p" =~ $allowed ]]; then bad="has a character outside the allowed set"
  else
    IFS=/ read -r -a segs <<<"$p"
    for s in "${segs[@]}"; do
      if [[ -z "$s" || "$s" == . || "$s" == .. ]]; then bad="has an empty, . or .. segment"; fi
    done
  fi
  if [[ -n "$bad" ]]; then
    echo "check-release-paths: $rel $bad; a release can hold $rule" >&2
    fail=1
  fi
done < <(find "$plugins" -path '*/templates/*' \( -type f -o -type l \) -print0)
exit "$fail"
