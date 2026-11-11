#!/usr/bin/env bash
# Cut the Latin preview subsets of the typeface plugins' full fonts into an albitor checkout's
# web/static/fonts/looks/ (ADR-0107 section 3). Deterministic: pinned fontTools (font-previews.pin),
# fixed Unicode set and flags (font-previews.json), fixed timestamps; running it twice gives
# byte-identical files. A subset is a Modified Version under the OFL, so a family with a Reserved
# Font Name is renamed (font-previews.json rfn_renames). Delivered apps never get subsets.
#
#   subset-previews.sh <path to an albitor-ltd/albitor checkout> [plugin...]
#   subset-previews.sh --out <dir> [plugin...]      write somewhere else (the drift test does)
#
# Needs python3 with venv, and network once to install the pinned tools into a cached venv
# (FONT_TOOLS_VENV overrides its location).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
plugins="${PLUGINS_DIR:-$root/plugins}"

# shellcheck disable=SC1091
source "$root/scripts/font-previews-lib.sh"

if [[ "${1:-}" == "--out" ]]; then
  out="${2:?usage: subset-previews.sh --out <dir> [plugin...]}"
  shift 2
else
  albitor="${1:?usage: subset-previews.sh <albitor checkout> [plugin...]}"
  shift
  [[ -d "$albitor/web/static/fonts" ]] || { echo "no web/static/fonts under $albitor" >&2; exit 2; }
  out="$albitor/web/static/fonts/looks"
fi

py="$(font_python)"
"$py" -I "$root/scripts/font-previews.py" generate "$plugins" "$out" "$@"
