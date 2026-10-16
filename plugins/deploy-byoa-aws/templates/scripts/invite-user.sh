#!/usr/bin/env bash
# Managed by Albitor (deploy-byoa-aws). Invites one person to the app's Cognito user pool, run by the
# invite workflow under the deploy identity.
#
#   scripts/invite-user.sh <user-pool-id> <email>
#
# The pool emails the person a username and a temporary password (the default invitation). At their
# first sign-in the app asks them to choose a password of their own (auth-cognito answers the
# NEW_PASSWORD_REQUIRED challenge), so nobody but them ever knows it. Inviting someone the pool
# already invited, and who has not signed in yet, sends the invitation again with a fresh temporary
# password. Someone who has already signed in is told so and changed in no way.
#
# This is the only place a user-pool admin call is made on the owner's behalf. The app's function
# holds none (the deploy identity can already create the pool, so this adds no authority).
set -uo pipefail

pool_id="${1:-}"
email="$(printf '%s' "${2:-}" | tr '[:upper:]' '[:lower:]')"

if ! printf '%s' "$pool_id" | grep -Eq '^[a-z]{2}-[a-z]+-[0-9]_[A-Za-z0-9]{1,32}$'; then
  echo "::error::this app has no user pool to invite anyone to (declare services.auth in albitor.yaml)" >&2
  exit 1
fi
# A plain address, and never one of the two reserved platform domains.
if ! printf '%s' "$email" | grep -Eq '^[a-z0-9._%+-]{1,64}@[a-z0-9-]+(\.[a-z0-9-]+)*\.[a-z]{2,}$' || [ "${#email}" -gt 254 ]; then
  echo "::error::the invitation needs a plain email address" >&2
  exit 1
fi
case "$email" in
  *.invalid)
    echo "::error::${email} is on a reserved domain that receives no mail" >&2
    exit 1
    ;;
esac
if ! command -v aws >/dev/null 2>&1; then
  echo "::error::the aws cli is required" >&2
  exit 1
fi

if status="$(aws cognito-idp admin-get-user --user-pool-id "$pool_id" --username "$email" \
  --query UserStatus --output text 2>/dev/null)"; then
  case "$status" in
    FORCE_CHANGE_PASSWORD)
      aws cognito-idp admin-create-user --user-pool-id "$pool_id" --username "$email" \
        --message-action RESEND >/dev/null || { echo "::error::could not resend the invitation" >&2; exit 1; }
      echo "invitation sent again to ${email}"
      exit 0
      ;;
    *)
      echo "${email} already has an account (${status}): nothing changed. They can use Forgot your password on the sign-in page."
      exit 0
      ;;
  esac
fi

out=""
if ! out="$(aws cognito-idp admin-create-user --user-pool-id "$pool_id" --username "$email" \
  --desired-delivery-mediums EMAIL \
  --user-attributes "Name=email,Value=${email}" "Name=email_verified,Value=true" 2>&1)"; then
  printf '%s\n' "$out" >&2
  echo "::error::could not invite ${email}" >&2
  exit 1
fi
echo "invitation sent to ${email}: they choose their own password at first sign-in"
