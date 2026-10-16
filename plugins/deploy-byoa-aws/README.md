# AWS deploy target plugin (`deploy-byoa-aws`)

The `deploy-target@1` plugin for an app in the customer's own AWS account
(albitor-ltd/albitor#3174, ADR-0090 section 6), split from the starter's `infrastructure/` and
`.github/`. A schema-2 plugin: `.claude-plugin/plugin.json` declares what it provides and
contributes, and the platform composer renders it into a new app.

## What it ships

| Repository path in the app | Mode | What |
| --- | --- | --- |
| `infrastructure/aws/{main,variables,versions,backend,outputs}.tf` | managed | The root: reads `albitor.yaml`, checks it, calls the modules |
| `infrastructure/aws/.terraform.lock.hcl` | managed | Provider checksums for the pinned providers |
| `infrastructure/aws/terraform.tfvars` | example | Per-environment choices: DNS mode, domain, alarm address |
| `.github/workflows/ci.yml` | managed | Build, test and Terraform checks; read-only |
| `.github/workflows/deploy.yml` | managed | Build, test, apply and ship under an OIDC-assumed role |
| `scripts/tf-unlock-stale.sh` | managed | Clears a stale same-account state lock before init |

It contributes the `github-actions` and `terraform` entries of the held `dependabot.yml`. The
modules in `modules/{data,api,site,alarms,auth}` are not copied into the app: the root consumes them by
`git::` source at `?ref=deploy-byoa-aws/v<version>`, so a root and its modules always match. The
ref is a literal, and a release moves it with `plugin.json`'s version (the tag is cut after merge;
a composed root cannot `init` before it exists).

## Behaviour worth knowing

- **One source of truth.** The root decodes `albitor.yaml` at plan time; nothing is generated per
  app. It renders `site`, `http`, `services.tables` and `services.auth`, and refuses a manifest
  that needs more (`services.objects`, `schedules`, `config`), rather than deploying without it.
- **The manifest is not trusted.** It is agent-written. Every value that steers a resource passes a
  variable `validation` in its module (names, key types, paths, hosts) and is used only as an
  attribute value, never spliced into a policy, script or command.
- **Pins are exact.** Terraform providers, the root's and modules' `required_providers`, the
  lockfile, Terraform itself and every workflow action (a full commit SHA).
- **No stored cloud key.** `deploy.yml` assumes the role in `env/<environment>.deploy.env` by OIDC.
  `id-token: write` is on the deploy job alone; triggers are `push` to main and `workflow_dispatch`
  (inputs `environment` and `ref`, as the platform dispatches it for a promotion). The `ship SPA`
  job runs after a green apply and ends with the `deploy-receipt` notice the platform reads.
  The platform writes the `env/` files and pins the role's trust to the repository and environment.
- **State** is the S3 backend with native locking; the bucket and key come from
  `env/<environment>.s3.tfbackend` at `init`.
- Deletion protection is on for every table and for the user pool.
- **`services.auth`** renders one Cognito user pool (the email address is the username), one public
  app client (`USER_PASSWORD_AUTH` and refresh, no secret, `PreventUserExistenceErrors` on, which
  the `auth-cognito` plugin requires), usernames case-insensitive, and one group per `groups`
  entry. `sign_up: invite_only` (the default) refuses public sign-up, and nothing on this target
  seeds an account, so the pool starts empty and **nobody can sign in until an operator creates a
  user** (below). `open` lets anyone register and confirm by email. `posture` is enforced by the app and changes nothing in the infrastructure. The function
  is handed `AUTH_USER_POOL_ID` and `AUTH_CLIENT_ID` (Lambda supplies `AWS_REGION`) and the root
  outputs both ids as `auth_user_pool_id` and `auth_client_id`. Mail goes through Cognito's default
  sender, which has a daily cap. Only `provider: cognito` is accepted; the module refuses the rest.

## Creating the first user (`invite_only`)

A user made by `AdminCreateUser` starts in `FORCE_CHANGE_PASSWORD`; the `auth-cognito` API answers
that state with 409 `challenge_unsupported`, and `ForgotPassword` is refused in it. So give the
user a permanent password at once, under your own AWS identity, with the email lowercased:

```bash
POOL=$(terraform -chdir=infrastructure/aws output -raw auth_user_pool_id)
aws cognito-idp admin-create-user --user-pool-id "$POOL" --username ada@example.com \
  --user-attributes Name=email,Value=ada@example.com Name=email_verified,Value=true \
  --message-action SUPPRESS
aws cognito-idp admin-set-user-password --user-pool-id "$POOL" --username ada@example.com \
  --password '<a password of 12 or more characters>' --permanent
```

This is a known gap: the target does not yet seed or invite users itself.

## Verification

`scripts/verify-deploy-byoa-aws.sh` (run by CI) format-checks the Terraform, runs each module's
`terraform test` and the root's tests against manifest fixtures, all with the AWS provider mocked:
no account, no credential. `--composed <dir>` does the same over a tree the composer wrote
(`scripts/compose-check.sh` passes one). `tests/compose` runs the real parser, composer, the
deploy-target@1 conformance suite and the workflow lint over this plugin.

## Not in this plugin

Previews, promotion between environments, schedules, object stores and runtime `config` are not
rendered yet; the stack refuses the manifest blocks at plan time.
