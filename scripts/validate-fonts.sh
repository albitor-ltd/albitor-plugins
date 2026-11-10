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
#   - role, style, weight and sha256 have the shapes core's fonts.go accepts;
#   - licence is OFL-1.1 and an OFL.txt sits beside the font file, its licence text (from the
#     "SIL OPEN FONT LICENSE Version 1.1" line on, whitespace-normalised) hashing to the canonical
#     value, and the text above it holding only copyright lines and the stock OFL boilerplate (an
#     added term is refused);
#   - rfn is "none" (the copyright line declares no Reserved Font Name) or the quoted names the
#     line declares ("Reserved Font Name 'X'", straight or curly quotes), comma-separated in rfn
#     when there are several ("Reserved Font Names 'A' and 'B'" is rfn "A, B"; each must match,
#     case-insensitively, and none may be left out);
#   - the file exists, starts with the woff2 signature (wOF2) and has the recorded sha256, and no
#     woff2 in the tree is missing from the block;
#   - every url() in the plugin's CSS is a relative or root-relative path that resolves to a file
#     (no scheme, //, @import, Google Fonts, or a data: URL over 1 KiB), and every @font-face that
#     loads a file says font-display: swap;
#   - no text file the plugin carries (CSS after comments are stripped; every other file except the
#     manifest, OFL.txt and prose docs) holds http:, https:, a protocol-relative //host or @import,
#     whatever function or bare string it sits in.
#
# tests/fonts/ holds the accepted and refused fixtures (scripts/test-fonts.sh).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command -v jq >/dev/null || { echo "jq is required" >&2; exit 2; }

# sha256 of the OFL-1.1 licence text (from the marker line on, normalised below). Checked when this
# gate was written against the OFL.txt of Source Sans 3, Nunito, Atkinson Hyperlegible Next, Inter
# and IBM Plex upstream; only Source Sans 3's is committed in this repository.
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

header_problem() { # header_problem <OFL.txt>: print why the text above the licence is not stock, if so
  local line pending_url=0
  while IFS= read -r line; do
    line="${line#"${line%%[![:space:]]*}"}"; line="${line%"${line##*[![:space:]]}"}"
    [[ -n "$line" ]] || continue
    if [[ $pending_url -eq 1 ]]; then
      pending_url=0
      case "$line" in http://scripts.sil.org/OFL | https://scripts.sil.org/OFL | https://openfontlicense.org | http://openfontlicense.org) continue ;; esac
      echo "unexpected line '$line'"; return
    fi
    case "$line" in
      -*) [[ -z "${line//-/}" ]] || { echo "unexpected line '$line'"; return; } ;;
      "This Font Software is licensed under the SIL Open Font License, Version 1.1.") ;;
      "This license is copied below, and is also available with a FAQ at:") pending_url=1 ;;
      "This license is copied below, and is also available with a FAQ at: "*)
        case "${line##* }" in http://scripts.sil.org/OFL | https://scripts.sil.org/OFL | https://openfontlicense.org | http://openfontlicense.org) ;;
          *) echo "unexpected line '$line'"; return ;; esac ;;
      Copyright\ *)
        if grep -qiE 'commercial|personal use|prohibit|forbid|restrict|additional|terms?[ :]' <<<"$line"; then
          echo "copyright line '$line' adds a condition"; return
        fi ;;
      *) echo "unexpected line '$line'"; return ;;
    esac
  done < <(ofl_header "$1")
  [[ $pending_url -eq 0 ]] || echo "FAQ line has no URL"
}

declared_rfns() { # the quoted names after "Reserved Font Name(s)" in the header, one per line
  ofl_header "$1" | sed -e "s/[‘’]/'/g" -e 's/[“”]/"/g' \
    | grep -iE 'reserved font names?' | sed -E 's/.*[Rr]eserved [Ff]ont [Nn]ames?//' | sed -E 's/\.( |$).*//' \
    | grep -oE "['\"][^'\"]+['\"]" | sed -E "s/^.(.*).\$/\1/" || true
}

scan_text() { # scan_text <relative name> <text>: remote origins and imports in any seeded text
  local name="$1" text="$2"
  if grep -qiE '@import' <<<"$text"; then err "$name: @import is not allowed"; fi
  if grep -qiE 'https?:' <<<"$text"; then err "$name: holds an http: or https: URL"; fi
  if grep -qE '(^|["'"'"'(=,[:space:]])//[A-Za-z0-9]' <<<"$text"; then err "$name: holds a protocol-relative URL"; fi
  if grep -qiE 'fonts\.(googleapis|gstatic)\.com|use\.typekit\.net|fonts\.adobe\.com|cdn\.jsdelivr\.net|unpkg\.com' <<<"$text"; then
    err "$name: references a third-party font host"
  fi
}

check_css() { # check_css <plugin dir> <css file>
  local pdir="$1" css="$2" crel text u path
  crel="${css#"$pdir"/}"
  text="$(sed -z 's#/\*[^*]*\*\+\([^/*][^*]*\*\+\)*/##g' "$css" | tr -d '\r')"
  scan_text "$crel" "$text"
  while IFS= read -r u; do
    [[ -n "$u" ]] || continue
    u="${u:4}"; u="${u%)}"
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
  done < <(grep -oiE 'url\([^)]*\)' <<<"$text" || true)
  # Every @font-face that loads a file must say font-display: swap.
  local n
  n="$(awk 'BEGIN{RS="@font-face"; bad=0} NR>1 { b=$0; sub(/\}.*/, "", b); if (b ~ /url\(/ && b !~ /font-display[[:space:]]*:[[:space:]]*swap/) bad++ } END{print bad}' <<<"${text,,}")"
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

    role="$(get role)"; style="$(get style)"; weight="$(get weight)"
    case "$role" in heading | body | data | label) ;; *) err "$face: role '$role' must be heading, body, data or label" ;; esac
    case "$style" in normal | italic) ;; *) err "$face: style '$style' must be normal or italic" ;; esac
    if ! [[ "$weight" =~ ^([1-9][0-9]{0,2}|1000)( ([1-9][0-9]{0,2}|1000))?$ ]]; then
      err "$face: weight '$weight' must be 1 to 1000 or a range such as \"400 700\""
    fi
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
      why="$(header_problem "$ofl")"
      if [[ -n "$why" ]]; then err "$face: OFL.txt header is not a copyright line and the stock boilerplate: $why"; fi
      mapfile -t declared < <(declared_rfns "$ofl" | tr '[:upper:]' '[:lower:]' | sort -u)
      if [[ "$rfn" == none ]]; then
        if ofl_header "$ofl" | grep -qi 'reserved font name'; then
          err "$face: rfn is none but the OFL.txt copyright line declares a Reserved Font Name"
        fi
      else
        mapfile -t want < <(tr ',' '\n' <<<"$rfn" | sed -E 's/^[[:space:]]+//; s/[[:space:]]+$//' | tr '[:upper:]' '[:lower:]' | sort -u)
        if [[ ${#declared[@]} -eq 0 || "${declared[*]}" != "${want[*]}" ]]; then
          err "$face: rfn '$rfn' does not match the quoted Reserved Font Name(s) in the OFL.txt copyright line (${declared[*]:-none declared})"
        fi
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

  # Every other text file the plugin carries, bar the manifest, the licence texts and prose docs.
  while IFS= read -r tf; do
    grep -Iq . "$tf" 2>/dev/null || continue
    scan_text "${tf#"$pdir"/}" "$(tr -d '\r' <"$tf")"
  done < <(find "$pdir" -type f ! -name '*.woff2' ! -name '*.css' ! -name 'OFL.txt' ! -name '*.md' \
    ! -name NOTICE ! -name 'LICEN[CS]E' ! -path '*/.claude-plugin/*' | sort)
done

[[ $fail -eq 0 ]] && echo "fonts: ${#dirs[@]} plugin(s) valid"
exit "$fail"
