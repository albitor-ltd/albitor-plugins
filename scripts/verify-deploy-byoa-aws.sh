#!/usr/bin/env bash
# Format-check, validate and test the deploy-byoa-aws plugin's Terraform with no AWS account: the
# AWS provider is mocked (terraform test), so nothing is created and no credential is read.
#
#   verify-deploy-byoa-aws.sh [--composed <dir>]
#
# Each module under plugins/deploy-byoa-aws/modules runs its own tests/*.tftest.hcl. The managed
# root (templates/infrastructure/aws) consumes the modules by a git source pinned to a release tag
# that does not exist until the plugin is tagged, so this lays the root over a scratch directory
# and points each module source at this checkout instead; the ref stays checked by the Go compose
# test. The root is then validated and run against the manifest fixtures in tests/root.
#
# --composed <dir> also validates and plans the root of a composed tree (the output of
# tests/compose, COMPOSE_OUT_BYOA): the real albitor.yaml the composer wrote, read by the rendered
# stack, with the same module rewrite. It runs the "composed" test only.
#
# Needs terraform >= 1.12 and network access to the Terraform registry for the provider plugins.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
plugin="$root/plugins/deploy-byoa-aws"
composed=""
case "${1:-}" in
  "") ;;
  --composed) composed="$(cd "${2:?--composed needs a directory}" && pwd)" ;;
  *) echo "usage: $0 [--composed <dir>]" >&2; exit 2 ;;
esac
command -v terraform >/dev/null || { echo "terraform is required" >&2; exit 2; }

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# terraform's plugin cache keeps repeated inits fast and offline-tolerant.
export TF_PLUGIN_CACHE_DIR="$work/plugin-cache"
mkdir -p "$TF_PLUGIN_CACHE_DIR"
export TF_IN_AUTOMATION=1

echo "== terraform fmt"
terraform fmt -check -recursive "$plugin/modules" "$plugin/templates/infrastructure/aws" "$plugin/tests"

for dir in "$plugin"/modules/*/; do
  name="$(basename "$dir")"
  echo "== module $name"
  scratch="$work/modules/$name"
  mkdir -p "$scratch"
  cp -R "$dir/." "$scratch/"
  (
    cd "$scratch"
    terraform init -backend=false -input=false >/dev/null
    # No terraform validate here: the site module declares a provider alias it is handed by its
    # caller, so it cannot validate alone. Its tests plan it, and the root's validate covers it.
    terraform test
  )
done

# lay_root <dest>: the managed root with each module source pointed at this checkout.
lay_root() {
  local dest="$1"
  mkdir -p "$dest"
  cp -R "$plugin/templates/infrastructure/aws/." "$dest/"
  # The backend is configured by the pipeline at init; there is none to reach here.
  rm -f "$dest/backend.tf"
  sed -i -E "s#\"git::https://github.com/albitor-ltd/albitor-plugins\.git//plugins/deploy-byoa-aws/modules/([a-z]+)\?ref=[^\"]+\"#\"$plugin/modules/\1\"#" "$dest/main.tf"
  if grep -q 'git::' "$dest/main.tf"; then
    echo "error: a module source in main.tf was not rewritten to the local checkout" >&2
    exit 1
  fi
}

if [[ -z "$composed" ]]; then
  echo "== root against the manifest fixtures"
  scratch="$work/root/infrastructure/aws"
  lay_root "$scratch"
  cp -R "$plugin/tests/root/." "$scratch/"
  # Fixtures are tests/root/fixtures; the test file sits beside them.
  rm -f "$scratch/terraform.tfvars"
  (
    cd "$scratch"
    terraform init -backend=false -input=false >/dev/null
    terraform validate
    terraform test -filter=root.tftest.hcl
  )
else
  echo "== composed tree: $composed"
  scratch="$work/composed"
  mkdir -p "$scratch"
  cp "$composed/albitor.yaml" "$scratch/albitor.yaml"
  lay_root "$scratch/infrastructure/aws"
  cp "$composed/infrastructure/aws/terraform.tfvars" "$scratch/infrastructure/aws/terraform.tfvars"
  mkdir -p "$scratch/infrastructure/aws/tests"
  cp "$plugin/tests/root/composed.tftest.hcl" "$scratch/infrastructure/aws/tests/composed.tftest.hcl"
  (
    cd "$scratch/infrastructure/aws"
    terraform init -backend=false -input=false >/dev/null
    terraform validate
    terraform test
  )
fi

echo "deploy-byoa-aws terraform verified"
