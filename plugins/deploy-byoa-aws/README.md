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
| `.github/workflows/deploy.yml` | managed | Build, test, apply, seed the platform accounts and ship under an OIDC-assumed role |
| `.github/workflows/invite.yml` | managed | Invites one person to the user pool; run by hand from the Actions tab |
| `scripts/tf-unlock-stale.sh` | managed | Clears a stale same-account state lock before init |
| `scripts/seed-platform-accounts.sh` | managed | Seeds the demo and verifier accounts into the pool, from the deploy job |
| `scripts/invite-user.sh` | managed | Sends one invitation; the invite workflow runs it |
| `docs/inviting-people.md` | example, held for handover | Tells the owner how to invite people; written into the repository at handover |

It contributes the `github-actions` and `terraform` entries of the held `dependabot.yml`. The
modules in `modules/{data,api,site,alarms,auth}` are not copied into the app: the root consumes them by
`git::` source at `?ref=deploy-byoa-aws/v<version>`, so a root and its modules always match. The
ref is a literal, and a release moves it with `plugin.json`'s version (the tag is cut after merge;
a composed root cannot `init` before it exists).

## Behaviour worth knowing

- **The site edge.** One response-headers policy of the module's own (HSTS, a same-origin
  Content-Security-Policy, frame DENY, nosniff, strict referrer) sits on the site and on `/api/*`.
  HSTS carries neither `includeSubDomains` nor `preload` unless `hsts_include_subdomains` or
  `hsts_preload` is set in `terraform.tfvars`: both commit hosts beyond this app and are the domain
  owner's call. There is no `custom_error_response`: `site.spa` is a viewer-request CloudFront
  Function on the site behaviour. Application routes serve the index, while `/api/...` and any path
  whose last segment has a dot pass through, so an unknown API path or a missing asset fails
  honestly instead of returning the shell. **A route whose last segment has a dot is therefore
  treated as a file**: `/users/jane.doe` reaches S3 and gets a 403. Keep dots out of the last
  segment of a client-side route (`/users/jane.doe/`, or an id in place of the name). Matching only
  known asset extensions was rejected: a real file with an unlisted extension would be rewritten to
  the shell.
- **Script hashes, not `'unsafe-inline'`.** The stack reads the HTML under `web/build` at plan time
  and admits its inline scripts by `sha256` hash, and refuses to plan a site whose build is absent
  rather than serve a CSP that blanks the app. A framework's inline bootstrap names its
  content-hashed chunks, so its hash changes every build, and the apply puts the new policy live
  before the bundle ships. The deploy workflow therefore reads the live stack's `csp_script_hashes`
  output (this build's hashes only) before the apply and hands it back as
  `csp_previous_script_hashes`: the policy admits the live build and the new one, and two builds
  at most. If the union would pass CloudFront's 1,783-character limit on the header, the previous
  build's hashes are dropped; a build whose own hashes pass it is refused at plan. The browser only
  calls its own origin (the API and sign-in go through `/api/*`), so `connect-src` is `'self'`; the
  `content_security_policy` variable replaces the whole policy.

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
  job runs after a green apply and ends with the `deploy-receipt` notice the platform reads: the
  app url, the commit, and for an app with sign-in the user pool id, the app client id and what the
  seed did (`seed_status`), which is how the platform knows which pool to sign in through.
  The platform writes the `env/` files and pins the role's trust to the repository and environment.
- **State** is the S3 backend with native locking; the bucket and key come from
  `env/<environment>.s3.tfbackend` at `init`.
- Deletion protection is on for every table and for the user pool.
- **`services.auth`** renders one Cognito user pool (the email address is the username), one public
  app client (`USER_PASSWORD_AUTH` and refresh, no secret, `PreventUserExistenceErrors` on, which
  the `auth-cognito` plugin requires), usernames case-insensitive, and one group per `groups`
  entry. `sign_up: invite_only` (the default) refuses public sign-up; the deploy seeds the two
  platform accounts and the owner invites everyone else (below), so the pool is never one nobody
  can sign in to. `open` lets anyone register and confirm by email. `posture` is enforced by the app
  and changes nothing in the infrastructure. The function
  is handed `AUTH_USER_POOL_ID` and `AUTH_CLIENT_ID` (Lambda supplies `AWS_REGION`) and the root
  outputs both ids as `auth_user_pool_id` and `auth_client_id`. Mail goes through Cognito's default
  sender, which has a daily cap. Only `provider: cognito` is accepted; the module refuses the rest.

## Who can sign in: the seeded accounts and the invite

**Seeded on every `dev` deploy** (Albitor's ADR-0071), after the apply and before the bundle ships,
under the deploy role: `demo@albitor-demo.invalid`, the owner's account for trying the app, and
`verifier@albitor-verify.invalid`, which Albitor's checks sign in with. Each is created confirmed with
a permanent password and no email. The passwords are Albitor's: it mints them and writes them to the
app's own secret store (`/albitor/apps/<segment>/secrets/ALBITOR_DEMO_ACCOUNT_PASSWORD` and
`ALBITOR_VERIFIER_ACCOUNT_PASSWORD`), where the deploy role reads them. `<segment>` is the app's slug,
which Albitor writes into `env/<environment>.deploy.env` as `ALBITOR_APP_SECRETS_SEGMENT`: it is not
the repository name, so the seed never guesses it. Without that line the seed fails the deploy
(`seed_status=failed:no_secrets_segment`) rather than skipping. Nothing here is in the app's
Terraform or its function: the function holds no user-pool admin action. An app with no
`services.auth` seeds nothing. A promotion to another environment seeds nothing, so a known account
never reaches a place real users depend on. A missing parameter is a warning annotation naming it,
not a failure, and the receipt says so (`seed_status=skipped_unprovisioned:<accounts>`); any other
read failure fails the deploy.

**Inviting people.** Run the `invite` workflow from the repository's Actions tab with the person's
email. The pool emails them a temporary password; at first sign-in the app asks them to choose their
own (the `auth-cognito` plugin answers the `NEW_PASSWORD_REQUIRED` challenge), so no one else ever
knows it. Running it again for someone who has not signed in yet resends the invitation; for someone
who has, it changes nothing. The temporary password is valid for 7 days. The email names the app and,
when the stack knows the address before the apply (a custom domain that is attached), where to sign
in. Running the workflow takes write access to the repository. The handover delivers
`docs/inviting-people.md`, which says all of this to the owner.

## Verification

`scripts/verify-deploy-byoa-aws.sh` (run by CI) format-checks the Terraform, runs each module's
`terraform test` and the root's tests against manifest fixtures, all with the AWS provider mocked:
no account, no credential. `--composed <dir>` does the same over a tree the composer wrote
(`scripts/compose-check.sh` passes one). `tests/compose` runs the real parser, composer, the
deploy-target@1 conformance suite and the workflow lint over this plugin.

## Not in this plugin

Previews, promotion between environments, schedules, object stores and runtime `config` are not
rendered yet; the stack refuses the manifest blocks at plan time.
