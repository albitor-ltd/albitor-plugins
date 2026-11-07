#!/usr/bin/env bash
# Prove check-internal-references.sh sees what the platform's handover check sees: a clean copy of
# a plugin passes, and each kind of reference planted into a rendered file fails it. A check that
# cannot fail would let the next header or tag that names the platform through.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
check="$root/scripts/check-internal-references.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

fresh() {
  rm -rf "$work/plugin"
  cp -R "$root/plugins/deploy-byoa-aws" "$work/plugin"
}

fresh
"$check" "$work/plugin" >/dev/null || { echo "FAIL: the unmodified plugin does not pass" >&2; exit 1; }

# plant <label> <rendered file> <line>: append the line and require a finding that names the file.
plant() {
  local label="$1" file="$2" line="$3" out
  fresh
  printf '%s\n' "$line" >>"$work/plugin/$file"
  if out="$("$check" "$work/plugin" 2>&1)"; then
    echo "FAIL: $label was not caught" >&2
    exit 1
  fi
  if ! grep -q "${file#templates/}\|${file}" <<<"$out"; then
    echo "FAIL: $label was caught but the finding does not name the file:" >&2
    echo "$out" >&2
    exit 1
  fi
  echo "ok: $label"
}

plant "a resource tag naming the platform" templates/infrastructure/aws/versions.tf '# Template = "albitor-deploy-byoa-aws"'
plant "an issue reference in a managed workflow" templates/.github/workflows/deploy.yml '# see #1234'
plant "an ADR id in a module" modules/site/main.tf '# per ADR-0041'
plant "the platform's name in a script" templates/scripts/invite-user.sh '# Albitor does this'
plant "the platform's name in the example tfvars" templates/infrastructure/aws/terraform.tfvars '# set by Albitor'
plant "a module source line with a comment" templates/infrastructure/aws/main.tf 'x = "git::https://github.com/albitor-ltd/albitor-plugins.git//plugins/p/modules/m?ref=v1" # albitor'

# --- a synthetic plugin: examples, declared globs, placeholders, every token ---------------------
mini() { # <manifest files json>
  rm -rf "$work/mini"
  mkdir -p "$work/mini/.claude-plugin" "$work/mini/templates/api/internal/store" \
    "$work/mini/skills/s/examples/api/internal"
  printf '{"name":"mini","contributes":{"files":%s}}\n' "$1" >"$work/mini/.claude-plugin/plugin.json"
  printf 'package store\n' >"$work/mini/templates/api/internal/store/a.go"
  printf 'package app\n' >"$work/mini/skills/s/examples/api/internal/b.go"
}
expect() { # <label> <pass|fail> (runs the check on $work/mini)
  local out rc=0
  out="$("$check" "$work/mini" 2>&1)" || rc=$?
  if [[ "$2" == pass && $rc -ne 0 ]] || [[ "$2" == fail && $rc -eq 0 ]]; then
    echo "FAIL: $1 (expected $2, exit $rc)" >&2; echo "$out" >&2; exit 1
  fi
  echo "ok: $1"
}

mini '[{"path":"api/internal/store/**","mode":"example"}]'
expect "a declared glob is expanded, not rejected" pass
printf '// Albitor owns this\n' >>"$work/mini/templates/api/internal/store/a.go"
expect "a reference in a file matched by a declared glob" fail

mini '[{"path":"api/internal/store","mode":"example"}]'
printf '// see #1234\n' >>"$work/mini/templates/api/internal/store/a.go"
expect "a reference in a file under a declared directory" fail

mini '[{"path":"api/missing.go","mode":"example"}]'
expect "an example-mode path with no template file is scanned via examples, not an error" pass
mini '[{"path":"api/missing.go","mode":"managed"}]'
expect "a managed path with no template file is an error" fail

mini '[]'
printf '// per ADR-0041\n' >>"$work/mini/skills/s/examples/api/internal/b.go"
expect "a reference in a skill example" fail

mini '[]'
printf 'var m = middlewareOf({{albitor.slot.api.middleware}})\n<html class="{{albitor.slot.web.root-classes.html}}" title="{{albitor.app_name}}">\n' \
  >>"$work/mini/skills/s/examples/api/internal/b.go"
expect "slot and app_name placeholders are stripped" pass
printf '// {{albitor.slot.x.html}} Albitor\n' >>"$work/mini/skills/s/examples/api/internal/b.go"
expect "a placeholder does not hide a reference beside it" fail
mini '[]'
printf '{{albitor.other}}\n' >>"$work/mini/skills/s/examples/api/internal/b.go"
expect "an unknown albitor placeholder is not stripped" fail

# Every token in the script: alone it is a resolved identifier and passes; beside a stray
# reference the line still fails, so the allowance covers the token and nothing around it.
eval "$(sed -n '/^tokens=(/,/^)/p' "$check")"
# shellcheck disable=SC2154 # tokens is assigned by the eval above
for token in "${tokens[@]}"; do
  mini '[]'
  printf 'x = "%sfoo"\n' "$token" >>"$work/mini/skills/s/examples/api/internal/b.go"
  expect "token $token passes alone" pass
  printf 'x = "%sfoo" // Albitor\n' "$token" >>"$work/mini/skills/s/examples/api/internal/b.go"
  expect "token $token does not hide a reference beside it" fail
done

# Regressions: case, spaces, empty globs and a broken manifest must not pass quietly.
mini '[]'
printf '// {{ALBITOR.SLOT.x}} and {{Albitor.app_name}}\n' >>"$work/mini/skills/s/examples/api/internal/b.go"
expect "an upper-case placeholder is not stripped (the composer would not substitute it)" fail

mini '[{"path":"api/with space/**","mode":"managed"}]'
mkdir -p "$work/mini/templates/api/with space"
printf '// see #1234\n' >"$work/mini/templates/api/with space/f.go"
expect "a declared glob whose path has a space is scanned" fail
printf 'package x\n' >"$work/mini/templates/api/with space/f.go"
expect "a clean file under a path with a space passes" pass

mini '[{"path":"nothing/**","mode":"managed"}]'
expect "a managed glob that matches nothing is an error" fail

mini '[{"path":"api/internal/store/**","mode":"managed"}]'
printf '{ not json' >"$work/mini/.claude-plugin/plugin.json"
expect "a malformed manifest is an error" fail

mini '[{"path":"nothing/**","mode":"example"}]'
rm -rf "$work/mini/skills"
expect "declared files with nothing at all to scan is an error" fail

# albitor-kit is an exact token: the layer name passes, a longer word does not.
mini '[]'
printf '@layer albitor-kit { :root { --a: 1; } }\n' >>"$work/mini/skills/s/examples/api/internal/b.go"
expect "the albitor-kit layer name passes" pass
printf '.albitor-kitchen { }\n' >>"$work/mini/skills/s/examples/api/internal/b.go"
expect "albitor-kit inside a longer word is a finding" fail

# A compliant plugin: neutral names, placeholders, tokens and a colour literal render clean.
mini '[{"path":"api/internal/store/**","mode":"example"}]'
cat >>"$work/mini/templates/api/internal/store/a.go" <<'GO'
// The store keeps one row per record.
var table = os.Getenv("ALBITOR_TABLE_NAME")
GO
printf 'color: #fff;\nvar r = routesOf({{albitor.slot.api.routes}})\n' >>"$work/mini/skills/s/examples/api/internal/b.go"
expect "compliant output passes" pass
echo "check-internal-references.sh verified"
