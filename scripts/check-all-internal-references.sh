#!/usr/bin/env bash
# Run check-internal-references.sh over every plugin.
set -uo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

rc=0
for dir in "$root"/plugins/*/; do
  "$root/scripts/check-internal-references.sh" "${dir%/}" || rc=1
done
exit "$rc"
