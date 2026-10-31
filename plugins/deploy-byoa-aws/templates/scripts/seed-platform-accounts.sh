#!/usr/bin/env bash
# Starting point (deploy-byoa-aws). Seeds the app's two platform accounts into its Cognito user
# pool, once per deploy, from the deploy job and never from the request-serving function.
#
#   scripts/seed-platform-accounts.sh <user-pool-id> <secrets-segment> [<admin-group> [<roles>]]
#
# Two accounts, one mechanism. The DEMO account is the app owner's: they sign
# in with it to try the app on handover day. The VERIFIER account is the platform's own: its checks sign
# in with it to prove the signed-in journeys work. Both are created confirmed, with a permanent
# password and no email (MessageAction SUPPRESS), so neither spends the default Cognito sender's
# daily quota and neither sits in FORCE_CHANGE_PASSWORD.
#
# Why here and not in the app: a seed that runs in the function needs cognito-idp:AdminCreateUser
# and AdminSetUserPassword on the whole pool for the role that answers anonymous traffic, which turns
# any request-path bug into a pool-wide account takeover. The deploy identity can already create the
# pool, so running the seed here adds no authority that did not exist.
#
# Each password was minted by the platform and written to the app's own secret store:
#   /albitor/apps/<secrets segment>/secrets/ALBITOR_DEMO_ACCOUNT_PASSWORD
#   /albitor/apps/<secrets segment>/secrets/ALBITOR_VERIFIER_ACCOUNT_PASSWORD
# The segment is what the platform wrote the parameters under (the app's slug). It
# is not derivable from the repository name, which can differ from the slug, so the platform writes
# it into env/<environment>.deploy.env as ALBITOR_APP_SECRETS_SEGMENT and deploy.yml passes it in.
# Without it the seed cannot know where to read, and fails (seed_status=failed:no_secrets_segment)
# rather than guessing: a guess would read as "no password" and skip silently.
# The usernames are constants below, not an input: nothing the app writes can name an identity or carry a password into here.
#
# Roles. An app has one kind of account unless it declares groups. When <admin-group> is given (the
# stack's resolved services.auth.admin_group, empty for an app with no groups) both accounts are
# added to it, and to no other group, so the owner's demo account can open the screens that group
# gates and the verifier has an identity that can walk them. AdminAddUserToGroup is idempotent, so
# a re-run is fine. $GITHUB_OUTPUT gets seeded_roles: the group when both accounts were seeded and added,
# else empty, and every role below whose account was seeded.
#
# One account per role. <roles> is the comma-separated list of every declared group
# (services.auth.groups). Each gets its own account, <role>@albitor-role.invalid, in that group and
# no other, so a spec signs in as exactly one role and the screens of every other role stay
# refused to it. The demo account is not given every group: it stays the owner's, in the admin group
# alone. The password is the app's own secret
#   /albitor/apps/<secrets segment>/secrets/ALBITOR_ROLE_<ROLE>_ACCOUNT_PASSWORD
# (the role upper-cased, hyphens as underscores), which the platform writes beside the other two. A role
# whose parameter is missing is a LOUD skip and is not in seeded_roles; it is never a failed deploy
# and never a seed_status, which keeps its three-account vocabulary. Auth@2's signInAs reads the same
# password, from outside the checkout, to sign in as the role (test-playwright, helpers/auth-v2.ts).
#
# Idempotent. It creates each account when it is absent and sets its password every time. A password
# reaches neither argv nor another step's environment: the set-password call reads a 0600 JSON file
# that is removed on the way out, and nothing is written to $GITHUB_ENV.
#
# No pool is a skip. A pool whose password parameter does not exist (ParameterNotFound) is a LOUD
# skip, a warning naming the account and the parameter, because the platform writes the parameter and a
# red deploy is not something the app's owner can fix. Any other read failure (access denied, KMS,
# throttling) is not "missing": it fails the step. $GITHUB_OUTPUT gets seed_status, one of
#   seeded | no_pool | skipped_unprovisioned:<demo|verifier>[,<demo|verifier>]
#   | failed:no_secrets_segment
set -uo pipefail

readonly demo_username="demo@albitor-demo.invalid"
readonly verifier_username="verifier@albitor-verify.invalid"
readonly demo_display_name="Demo User"

pool_id="${1:-}"
secrets_segment="${2:-}"
admin_group="${3:-}"
roles_csv="${4:-}"

# seeded_roles is the group the accounts were added to, or empty. It is written beside every
# seed_status so a reader never has to tell "absent" from "none".
seeded_roles=""

write_status() {
  if [ -n "${GITHUB_OUTPUT:-}" ]; then
    echo "seed_status=$1" >>"$GITHUB_OUTPUT"
    echo "seeded_roles=${seeded_roles}" >>"$GITHUB_OUTPUT"
  fi
}

if [ -z "$pool_id" ]; then
  echo "no user pool for this app: nothing to seed"
  write_status no_pool
  exit 0
fi
if ! printf '%s' "$pool_id" | grep -Eq '^[a-z]{2}-[a-z]+-[0-9]_[A-Za-z0-9]{1,32}$'; then
  echo "::error::the user pool id is not a pool id" >&2
  exit 1
fi
if ! command -v aws >/dev/null 2>&1 || ! command -v python3 >/dev/null 2>&1; then
  echo "::error::the aws cli and python3 are required to seed the platform accounts" >&2
  exit 1
fi

if [ -n "$admin_group" ] && ! printf '%s' "$admin_group" | grep -Eq '^[a-z][a-z0-9-]{1,31}$'; then
  echo "::error::the admin group is not a group name" >&2
  exit 1
fi

roles=()
if [ -n "$roles_csv" ]; then
  IFS=',' read -r -a roles <<<"$roles_csv"
  for role in "${roles[@]}"; do
    if ! printf '%s' "$role" | grep -Eq '^[a-z][a-z0-9-]{1,31}$'; then
      echo "::error::a role is not a group name" >&2
      exit 1
    fi
  done
fi

# The parameter path segment is the one the platform wrote the parameters under, handed in by the
# workflow from the platform-written deploy.env. Absent is a failure, not a skip.
if [ -z "$secrets_segment" ]; then
  echo "::error::env/<environment>.deploy.env carries no ALBITOR_APP_SECRETS_SEGMENT, so the seed cannot tell where the platform wrote the account passwords. Redeploy after the platform rewrites that file" >&2
  write_status failed:no_secrets_segment
  exit 1
fi
if ! printf '%s' "$secrets_segment" | grep -Eq '^[A-Za-z0-9._-]{1,100}$'; then
  echo "::error::the secrets segment is not a plain token" >&2
  write_status failed:no_secrets_segment
  exit 1
fi

# `payload` is empty until an account gets far enough to write one, and `rm -f ""` is a no-op, so
# the trap is right on every exit path.
payload=""
read_err="$(mktemp)"
trap 'rm -f "$payload" "$read_err"' EXIT

unprovisioned=()
added_to_group=()
seeded_role_accounts=()

# seed_account <label> <short name> <username> <parameter name> <display name> <group> <track>
# <group> is the one group the account joins ("" for none). <track> is "yes" for the demo and the
# verifier, whose absence is a seed_status; a role account's absence is only a warning.
seed_account() {
  local label="$1" short="$2" username="$3" param_name="$4" display_name="${5:-}" group="${6:-}" track="${7:-yes}"
  local param="/albitor/apps/${secrets_segment}/secrets/${param_name}"
  local password read_status create_out
  local -a attributes

  # Unquoted on purpose: an assignment never word-splits. stderr goes to a file, not /dev/null: only
  # a missing parameter is a skip, and the error text is how the two are told apart.
  password=$(aws ssm get-parameter --name "$param" --with-decryption \
    --query Parameter.Value --output text 2>"$read_err")
  read_status=$?
  if [ "$read_status" -ne 0 ]; then
    if ! grep -q 'ParameterNotFound' "$read_err"; then
      cat "$read_err" >&2
      echo "::error::could not read ${param} for the ${label}: it is not missing, so this is not a skip. Check that the deploy role may read it and decrypt it, then redeploy" >&2
      return 1
    fi
    password=""
  fi
  if [ -z "$password" ] || [ "$password" = "None" ] || [ "$password" = "__albitor_pending__" ]; then
    echo "::warning title=seed-skipped::the app has user pool ${pool_id} but ${param} holds no password, so the ${label} is not seeded"
    if [ "$track" = "yes" ]; then
      unprovisioned+=("$short")
    fi
    return 0
  fi

  attributes=("Name=email,Value=${username}" "Name=email_verified,Value=true")
  if [ -n "$display_name" ]; then
    attributes+=("Name=name,Value=${display_name}")
  fi
  if ! create_out="$(aws cognito-idp admin-create-user \
    --user-pool-id "$pool_id" --username "$username" \
    --message-action SUPPRESS --user-attributes "${attributes[@]}" 2>&1)"; then
    if ! printf '%s' "$create_out" | grep -q 'UsernameExistsException'; then
      printf '%s\n' "$create_out" >&2
      echo "::error::could not create the ${label} in ${pool_id}" >&2
      return 1
    fi
    echo "the ${label} already exists: resetting its password"
  fi

  payload="$(mktemp)"
  chmod 600 "$payload"
  # python3 writes the JSON because a password may hold any character a here-document would have to
  # escape; the values go through its environment, which /proc shows only to the owner.
  if ! SEED_POOL_ID="$pool_id" SEED_USERNAME="$username" SEED_PASSWORD="$password" \
    python3 -c 'import json, os, sys
json.dump({
    "UserPoolId": os.environ["SEED_POOL_ID"],
    "Username": os.environ["SEED_USERNAME"],
    "Password": os.environ["SEED_PASSWORD"],
    "Permanent": True,
}, open(sys.argv[1], "w"))' "$payload"; then
    echo "::error::could not write the set-password request" >&2
    return 1
  fi
  if ! aws cognito-idp admin-set-user-password --cli-input-json "file://${payload}" >/dev/null; then
    echo "::error::could not set the ${label}'s password in ${pool_id}" >&2
    return 1
  fi
  rm -f "$payload"
  payload=""
  if [ -n "$group" ]; then
    if ! aws cognito-idp admin-add-user-to-group \
      --user-pool-id "$pool_id" --username "$username" --group-name "$group" >/dev/null; then
      echo "::error::could not add the ${label} to the ${group} group in ${pool_id}" >&2
      return 1
    fi
    if [ "$track" = "yes" ]; then
      added_to_group+=("$short")
    else
      seeded_role_accounts+=("$group")
    fi
    echo "the ${label} is in the ${group} group"
  fi
  echo "the ${label} is seeded in ${pool_id}"
}

# Both are attempted whichever fails: they are independent fixtures, and a reader of a red deploy is
# owed both findings rather than the first.
status=0
seed_account "demo account" demo "$demo_username" ALBITOR_DEMO_ACCOUNT_PASSWORD "$demo_display_name" "$admin_group" yes || status=1
seed_account "verifier account" verifier "$verifier_username" ALBITOR_VERIFIER_ACCOUNT_PASSWORD "" "$admin_group" yes || status=1
for role in ${roles[@]+"${roles[@]}"}; do
  upper="$(printf '%s' "$role" | tr 'a-z-' 'A-Z_')"
  seed_account "${role} role account" "role-${role}" "${role}@albitor-role.invalid" \
    "ALBITOR_ROLE_${upper}_ACCOUNT_PASSWORD" "" "$role" no || status=1
done

# seeded_roles claims a role only when both accounts were seeded and added to it: a skipped account
# means a role-gated journey may have no identity, and the receipt must not say otherwise.
if [ "$status" -eq 0 ] && [ "${#unprovisioned[@]}" -eq 0 ] && [ "${#added_to_group[@]}" -eq 2 ]; then
  seeded_roles="$admin_group"
fi
# Each role account seeded and added to its own group claims that role, once.
for role in ${seeded_role_accounts[@]+"${seeded_role_accounts[@]}"}; do
  case ",${seeded_roles}," in
    *",${role},"*) ;;
    *) seeded_roles="${seeded_roles:+${seeded_roles},}${role}" ;;
  esac
done

if [ "${#unprovisioned[@]}" -gt 0 ]; then
  write_status "skipped_unprovisioned:$(IFS=,; echo "${unprovisioned[*]}")"
else
  write_status seeded
fi
exit "$status"
