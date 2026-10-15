#!/usr/bin/env bash
#
# tf-unlock-stale.sh — clear a *stale, same-account* Terraform S3 state lock.
#
# A killed custom-domain apply once left an S3 state lock orphaned, and every
# later deploy failed "Error acquiring the state lock" for ~19h because clearing
# it needed an operator. This guard runs at the top of the
# deploy job and clears a lock only when it is BOTH:
#   - same-account: the lock object lives in this account's own state bucket
#     (albitor-tfstate-<account_id>), verified against sts:GetCallerIdentity, so
#     we can never clear another account's lock; and
#   - stale: the lock object is older than LOCK_STALE_MINUTES, so we never race a
#     genuinely-concurrent apply (those are handled by -lock-timeout instead).
#
# A fresh lock (younger than the threshold) is left alone — exit 0, let the apply
# wait it out. No lock present — exit 0.
#
# Usage:
#   tf-unlock-stale.sh <state-bucket> <state-key> [stale-minutes]
# Env: AWS credentials in the environment (the deploy identity).
set -euo pipefail

BUCKET="${1:?usage: tf-unlock-stale.sh <bucket> <key> [stale-minutes]}"
KEY="${2:?usage: tf-unlock-stale.sh <bucket> <key> [stale-minutes]}"
STALE_MINUTES="${3:-${LOCK_STALE_MINUTES:-30}}"
LOCK_KEY="${KEY}.tflock"

log() { printf '[tf-unlock-stale] %s\n' "$*" >&2; }

# --- same-account guard ------------------------------------------------------
account_id="$(aws sts get-caller-identity --query Account --output text)"
expected_bucket="albitor-tfstate-${account_id}"
if [[ "${BUCKET}" != "${expected_bucket}" ]]; then
  log "refusing to auto-unlock: bucket '${BUCKET}' is not this account's state bucket ('${expected_bucket}'). Clear it manually if you are sure."
  exit 0
fi

# --- is there a lock at all? -------------------------------------------------
if ! last_modified="$(aws s3api head-object --bucket "${BUCKET}" --key "${LOCK_KEY}" \
  --query LastModified --output text 2>/dev/null)"; then
  log "no state lock present (${LOCK_KEY}); nothing to do."
  exit 0
fi

# --- stale? ------------------------------------------------------------------
lock_epoch="$(date -u -d "${last_modified}" +%s 2>/dev/null || date -u -jf '%Y-%m-%dT%H:%M:%S%z' "${last_modified}" +%s 2>/dev/null || echo 0)"
now_epoch="$(date -u +%s)"
age_minutes=$(( (now_epoch - lock_epoch) / 60 ))

if (( lock_epoch == 0 )); then
  log "could not parse lock timestamp '${last_modified}'; refusing to auto-unlock for safety."
  exit 0
fi

if (( age_minutes < STALE_MINUTES )); then
  log "lock is ${age_minutes}m old (< ${STALE_MINUTES}m): treating as live, not clearing. -lock-timeout will wait it out."
  exit 0
fi

# --- clear it ----------------------------------------------------------------
holder="$(aws s3api get-object --bucket "${BUCKET}" --key "${LOCK_KEY}" /dev/stdout 2>/dev/null | tr -d '\n' || true)"
log "clearing STALE lock ${age_minutes}m old (threshold ${STALE_MINUTES}m). Holder: ${holder:-<unreadable>}"
aws s3api delete-object --bucket "${BUCKET}" --key "${LOCK_KEY}"
log "stale lock cleared."
