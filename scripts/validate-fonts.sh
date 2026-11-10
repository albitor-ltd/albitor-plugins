#!/usr/bin/env bash
# The font self-hosting gate (ADR-0107 section 4) over a typeface plugin's tree. Fails closed.
#
#   validate-fonts.sh [plugin-dir...]   (default: every plugins/* that provides typeface@1 or
#                                        states a fonts block)
#
# For each plugin it checks:
#   - a plugin providing typeface@1 states a non-empty `fonts` block, and only such a plugin does;
#   - each entry is closed and carries family, role, weight, style, file, licence, sha256, source
#     and rfn (rfn is a name or "none"; an unrecorded RFN status is refused);
#   - licence is OFL-1.1 and an OFL.txt sits beside the font file, its licence text (from the
#     "SIL OPEN FONT LICENSE Version 1.1" line on, whitespace-normalised) hashing to the canonical
#     value, with the recorded rfn agreeing with the copyright line above the text;
#   - the file exists, starts with the woff2 signature (wOF2) and has the recorded sha256, and no
#     woff2 in the tree is missing from the block;
#   - every url() in the plugin's CSS is a relative or root-relative path that resolves to a file
#     (no scheme, //, @import, Google Fonts, or a data: URL over 1 KiB), and every @font-face that
#     loads a file says font-display: swap.
#
# tests/fonts/ holds the accepted and refused fixtures (scripts/test-fonts.sh).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command -v jq >/dev/null || { echo "jq is required" >&2; exit 2; }

# sha256 of the OFL-1.1 licence text common to every family's OFL.txt (verified against Source
# Sans 3, Nunito, Atkinson Hyperlegible Next, Inter and IBM Plex).
canonical_ofl_sha256=353d6b1e70fd27a8008c626965f98740f784359d93b4c85b769573dbf6295bdc
ofl_marker='^SIL OPEN FONT LICENSE Version 1.1 - 26 February 2007'
max_data_url=1024

if [[ $# -gt 0 ]]; then
  dirs=("$@")
else
  dirs=()
  shopt -s nullglob
  for m in "$root"/plugins/*/.claude-plugin/plugin.json; do
    if jq -e '((.provides // []) | any(startswith("typeface@"))) or has("fonts")' "$m" >/dev/null 2>&1; then
      dirs+=("$(dirname "$(dirname "$m")")")
    fi
  done
fi
[[ ${#dirs[@]} -gt 0 ]] || { echo "no typeface plugins found" >&2; exit 2; }

fail=0
err() { echo "error: $rel: $*" >&2; fail=1; }

ofl_hash() { # the licence text of an OFL.txt, normalised, hashed
  tr -d '\r' <"$1" | sed -n "/$ofl_marker/,\$p" | sed -E 's/[[:space:]]+$//' | sed '/^$/d' | sha256sum | cut -d' ' -f1
}

ofl_header() { # the copyright notice above the licence text
  tr -d '\r' <"$1" | sed "/$ofl_marker/,\$d"
}

check_css() { # check_css <plugin dir> <css file>
  local pdir="$1" css="$2" crel text u path
  crel="${css#"$pdir"/}"
  text="$(sed -z 's#/\*[^*]*\*\+\([^/*][^*]*\*\+\)*/##g' "$css" | tr -d '\r')"
  if grep -qiE '@import' <<<"$text"; then err "$crel: @import is not allowed"; fi
  if grep -qiE 'fonts\.(googleapis|gstatic)\.com|use\.typekit\.net|fonts\.adobe\.com|cdn\.jsdelivr\.net|unpkg\.com' <<<"$text"; then
    err "$crel: references a third-party font host"
  fi
  while IFS= read -r u; do
    [[ -n "$u" ]] || continue
    u="${u#url(}"; u="${u%)}"
    u="${u#"${u%%[![:space:]]*}"}"; u="${u%"${u##*[![:space:]]}"}"
    u="${u#[\"\']}"; u="${u%[\"\']}"
    case "$u" in
      data:*)
        if [[ ${#u} -gt $max_data_url ]]; then err "$crel: data: URL larger than $max_data_url bytes"; fi ;;
      //*) err "$crel: protocol-relative URL '$u'" ;;
      /*) ;; # root-relative: resolved against the app's output, which the build gate checks
      *:*) err "$crel: absolute URL '$u'" ;;
      *)
        path="$(dirname "$css")/${u%%[?#]*}"
        [[ -f "$path" ]] || err "$crel: url '$u' does not resolve to a file" ;;
    esac
  done < <(grep -oE 'url\([^)]*\)' <<<"$text" || true)
  # Every @font-face that loads a file must say font-display: swap.
  local n
  n="$(awk 'BEGIN{RS="@font-face"; bad=0} NR>1 { b=$0; sub(/\}.*/, "", b); if (b ~ /url\(/ && b !~ /font-display[[:space:]]*:[[:space:]]*swap/) bad++ } END{print bad}' <<<"$text")"
  if [[ "$n" != 0 ]]; then err "$crel: $n @font-face rule(s) without font-display: swap"; fi
}

for pdir in "${dirs[@]}"; do
  pdir="${pdir%/}"
  rel="${pdir#"$root"/}"
  manifest="$pdir/.claude-plugin/plugin.json"
  if ! jq -e . "$manifest" >/dev/null 2>&1; then err "plugin.json missing or not valid JSON"; continue; fi
  provides="$(jq -r '((.provides // []) | any(startswith("typeface@")))' "$manifest")"
  has_fonts="$(jq -r 'has("fonts")' "$manifest")"
  if [[ "$provides" != true ]]; then
    if [[ "$has_fonts" == true ]]; then err "fonts is stated but the plugin does not provide typeface@1"; fi
    continue
  fi
  if [[ "$has_fonts" != true ]]; then err "provides typeface@1 but states no fonts block"; continue; fi
  if ! jq -e '.fonts | type == "array" and length > 0 and length <= 32' "$manifest" >/dev/null; then
    err "fonts must be a non-empty array of at most 32 faces"; continue
  fi

  listed="$(mktemp)"
  count="$(jq '.fonts | length' "$manifest")"
  for ((i = 0; i < count; i++)); do
    face="fonts[$i]"
    if ! jq -e ".fonts[$i] | type == \"object\"" "$manifest" >/dev/null; then err "$face: must be an object"; continue; fi
    extra="$(jq -r ".fonts[$i] | keys_unsorted[] | select(IN(\"family\",\"role\",\"weight\",\"style\",\"file\",\"licence\",\"sha256\",\"source\",\"rfn\") | not)" "$manifest")"
    if [[ -n "$extra" ]]; then err "$face: unknown key(s): $(tr '\n' ' ' <<<"$extra")"; fi
    missing=0
    for k in family role weight style file licence sha256 source rfn; do
      v="$(jq -r ".fonts[$i].$k | if type == \"string\" then . elif type == \"number\" then tostring else \"\" end" "$manifest")"
      if [[ -z "${v//[[:space:]]/}" ]]; then err "$face: $k is required"; missing=1; fi
    done
    [[ $missing -eq 0 ]] || continue
    get() { jq -r ".fonts[$i].$1 | tostring" "$manifest"; }
    file="$(get file)"; licence="$(get licence)"; sha="$(get sha256)"; rfn="$(get rfn)"

    if [[ "$licence" != "OFL-1.1" ]]; then err "$face: licence '$licence' is not OFL-1.1"; fi
    if ! [[ "$sha" =~ ^[0-9a-f]{64}$ ]]; then err "$face: sha256 must be 64 lowercase hex characters"; fi
    case "$file" in
      /* | *..* | ./* | */ | *//* ) err "$face: file '$file' is not a clean relative path"; continue ;;
      *.woff2) ;;
      *) err "$face: file '$file' is not a .woff2 file"; continue ;;
    esac
    echo "$file" >>"$listed"
    path="$pdir/$file"
    if [[ ! -f "$path" ]]; then err "$face: file '$file' does not exist"; continue; fi
    if [[ "$(head -c4 "$path")" != "wOF2" ]]; then
      err "$face: '$file' does not begin with the woff2 signature (wOF2)"
    elif [[ "$(sha256sum "$path" | cut -d' ' -f1)" != "$sha" ]]; then
      err "$face: '$file' sha256 differs from the manifest"
    fi

    ofl="$(dirname "$path")/OFL.txt"
    if [[ ! -f "$ofl" ]]; then
      err "$face: no OFL.txt beside '$file'"
    else
      if [[ "$(ofl_hash "$ofl")" != "$canonical_ofl_sha256" ]]; then
        err "$face: OFL.txt beside '$file' does not match the canonical OFL-1.1 text"
      fi
      header="$(ofl_header "$ofl")"
      if [[ "$rfn" == none ]]; then
        if grep -qi 'reserved font name' <<<"$header"; then
          err "$face: rfn is none but the OFL.txt copyright line declares a Reserved Font Name"
        fi
      elif ! grep -qi 'reserved font name' <<<"$header" || ! grep -qF -- "$rfn" <<<"$header"; then
        err "$face: rfn '$rfn' is not declared in the OFL.txt copyright line"
      fi
    fi
  done

  # A font file with no manifest line fails.
  while IFS= read -r f; do
    f="${f#"$pdir"/}"
    if ! grep -qxF -- "$f" "$listed"; then err "$f: woff2 file has no fonts entry"; fi
  done < <(find "$pdir" -type f -name '*.woff2' | sort)
  rm -f "$listed"

  while IFS= read -r css; do
    check_css "$pdir" "$css"
  done < <(find "$pdir" -type f -name '*.css' | sort)
done

[[ $fail -eq 0 ]] && echo "fonts: ${#dirs[@]} plugin(s) valid"
exit "$fail"
