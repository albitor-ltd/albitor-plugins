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
echo "check-internal-references.sh verified"
