#!/usr/bin/env bash
# Run the delivered-tree internal-references check over what a plugin renders.
#
#   check-internal-references.sh <plugin-dir>
#
# A file a plugin renders into a customer's repository is read by people with no access to the
# platform, so it must not cite an issue, an ADR or the platform by name, even though the platform
# exempts its own managed files from the scan: no agent is ever pushed to edit one. This mirrors
# the platform's handover gate (albitor-ltd/albitor, app/runner/gates/handover_reference.sh, the
# `handover-clean/no-internal-references` floor check); that script is the source of truth. Keep
# the patterns, the token list and the shapes below in step with it. The platform's per-app
# exemptions (the repository's own owner/repo slug) are not mirrored:
# they do not apply to a plugin. The two composer placeholders, {{albitor.slot.<name>}} and
# {{albitor.app_name}}, are stripped for a different reason: the platform scans the composed tree,
# where the composer has already replaced them, but this check reads the plugin's pre-composition
# text, where they still stand. They are matched case-sensitively and in the composer's own shape
# (placeholderRE and tokenPattern in app/core), so a mis-cased one, which the composer would leave
# in place, is still a finding.
#
# It checks the files the manifest's `contributes.files` renders (templates/<path>; a declared glob
# or directory is expanded), every file under the plugin's skills/*/examples/ (an agent copies
# them into the customer's repository), and the Terraform modules the rendered stack fetches
# (modules/), which the customer's plan downloads.
# Exit 0: clean. Exit 2: a finding, printed as <path>:<line>: <match>: <text>.
set -uo pipefail
export LC_ALL=C

plugin="${1:?usage: check-internal-references.sh <plugin-dir>}"
manifest="$plugin/.claude-plugin/plugin.json"
[[ -f "$manifest" ]] || { echo "error: $manifest not found" >&2; exit 2; }
command -v jq >/dev/null || { echo "jq is required" >&2; exit 2; }
jq -e . "$manifest" >/dev/null || { echo "error: $manifest is not valid JSON" >&2; exit 2; }

# Literal identifiers the delivered pipeline resolves at run time (lower case; the scan folds case).
tokens=(
  albitor-tfstate- /albitor/apps/
  albitor_demo_account_password albitor_verifier_account_password albitor_app_secrets_segment
  albitor_journey_ albitor_local albitor_table_ albitor_queue_ albitor_secrets_parameter_prefix
  albitor_role_ __albitor_pending__
  albitor_owner_email albitor_owner_temp_password albitor_verifier_b_account_password
  albitor-demo.invalid albitor-verify.invalid albitor-role.invalid
  albitor_services_url albitor_bucket_ albitor_objects_path_
  '<!-- albitor:home-stub -->'
)

# files: "<disk path>|<rel path in the customer's repository>"
files=()
declare -A seen=()
add_file() { # <disk> <rel>
  [[ -n "${seen[$1]:-}" ]] && return
  seen["$1"]=1
  files+=("$1|$2")
}
while IFS=$'\t' read -r rel mode; do
  if [[ "$rel" == *[\*\?\[]* || -d "$plugin/templates/$rel" ]]; then
    # A declared glob or directory: scan whatever it matches under templates/.
    matched=0
    while IFS= read -r -d '' f; do
      add_file "$f" "${f#"$plugin/templates/"}"
      matched=$((matched + 1))
    done < <(
      cd "$plugin/templates" 2>/dev/null || exit 0
      shopt -s globstar nullglob dotglob
      IFS=   # no word splitting: a path with a space is one pattern; globbing still applies
      pat="$rel"
      [[ -d "$pat" ]] && pat="${pat%/}/**"
      for m in $pat; do [[ -f "$m" ]] && printf '%s\0' "$plugin/templates/$m"; done | sort -z
    )
    if [[ $matched -eq 0 && "$mode" != "example" ]]; then
      echo "error: $rel is declared but matches no file under templates/" >&2; exit 2
    fi
  elif [[ -f "$plugin/templates/$rel" ]]; then
    add_file "$plugin/templates/$rel" "$rel"
  elif [[ "$mode" != "example" ]]; then
    echo "error: $rel is declared but templates/$rel is missing" >&2; exit 2
  fi
done < <(jq -r '.contributes.files[]? | [.path, (.mode // "")] | @tsv' "$manifest")
# Examples are copied into the customer's repository by the agent; they are scanned in full.
while IFS= read -r -d '' f; do
  ex="${f#"$plugin"/skills/}"
  add_file "$f" "${ex#*/examples/}"
done < <(find "$plugin"/skills -path '*/examples/*' -type f -print0 2>/dev/null | sort -z)
if [[ -d "$plugin/modules" ]]; then
  while IFS= read -r -d '' f; do
    case "$f" in */tests/*) continue ;; esac
    add_file "$f" "${f#"$plugin"/}"
  done < <(find "$plugin/modules" -type f -print0 | sort -z)
fi

declared="$(jq -r '(.contributes.files // []) | length' "$manifest")"
if [[ "$declared" -gt 0 && ${#files[@]} -eq 0 ]]; then
  echo "error: $manifest declares $declared files but none was found to scan" >&2; exit 2
fi

findings=""
for entry in "${files[@]}"; do
  disk="${entry%%|*}"
  rel="${entry#*|}"
  grep -qI . "$disk" 2>/dev/null || continue
  while IFS= read -r hit; do
    line="${hit%%:*}"
    text="${hit#*:}"
    # shellcheck disable=SC2018,SC2019 # ASCII-only case fold is intended, as in the platform's check
    # Composer placeholders (case-sensitive, the composer's own shapes) go before the case fold.
    cleaned="$(printf '%s' "$text" | sed -E 's/\{\{albitor\.(slot\.[A-Za-z0-9_.-]+|app_name)\}\}/ /g' | tr 'A-Z' 'a-z')"

    for token in "${tokens[@]}"; do
      cleaned="${cleaned//"$token"/ }"
    done
    # The kit's cascade layer, as an exact token (not a substring), as in the platform's check.
    cleaned="$(printf '%s' "$cleaned" | sed -E 's#(^|[^a-z0-9_-])(albitor-kit)([^a-z0-9_-]|$)#\1 \3#g')"
    # albitor.yaml as a bare file name.
    # shellcheck disable=SC2016 # a sed regex: the $ is literal
    cleaned="$(printf '%s' "$cleaned" | sed -E 's#(^|[[:space:]"(=`])albitor\.yaml(\.?([[:space:]"),:;`]|$))#\1 \2#g')"
    # The managed stack's two shapes, anchored to its main.tf.
    if [[ "$rel" == "infrastructure/aws/main.tf" ]]; then
      cleaned="$(printf '%s' "$cleaned" | sed -E -e 's#"albitor/v1"# #g' -e 's#^[[:space:]]*source[[:space:]]*=[[:space:]]*"git::https://github\.com/albitor-ltd/albitor-plugins\.git//plugins/[a-z0-9-]+/modules/[a-z0-9-]+\?ref=[a-z0-9./_-]+"[[:space:]]*$# #')"
    fi
    # A manifest version in the CI workflow's check.
    [[ "$rel" == ".github/workflows/ci.yml" ]] && cleaned="$(printf '%s' "$cleaned" | sed -E 's#"albitor/v1"# #g')"
    # A colour literal in value position, or as a var() fallback, is not an issue reference.
    cleaned="$(printf '%s' "$cleaned" | sed -E "s/[:=][[:space:]]*['\"]?#[0-9a-f]+/ /g")"
    cleaned="$(printf '%s' "$cleaned" | sed -E 's/var\([^,)]*,[[:space:]]*#[0-9a-f]+/ /g')"

    match="$(printf '%s' "$cleaned" | grep -oEi '#[0-9]{3,}\b|adr-[0-9]{4}|albitor' | head -n 1 || true)"
    [[ -n "$match" ]] || continue
    findings="${findings}${rel}:${line}: ${match}: ${text:0:160}
"
  done < <(grep -nEi '#[0-9]{3,}\b|adr-[0-9]{4}|albitor' "$disk" 2>/dev/null || true)
done

if [[ -z "$findings" ]]; then
  echo "internal-references check passed: ${#files[@]} files rendered or fetched by $(jq -r .name "$manifest")"
  exit 0
fi
echo "internal-references check FAILED: a rendered file names the platform, an issue or an ADR." >&2
echo "A customer's repository is read by people with no access to any of them, and the platform's" >&2
echo "own handover check would send an agent to edit the file. Keep the reason, drop the reference." >&2
echo >&2
printf '%s' "$findings" >&2
exit 2
