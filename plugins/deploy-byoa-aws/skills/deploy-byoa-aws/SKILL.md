---
name: deploy-byoa-aws
description: Use when an app composed with the deploy-byoa-aws plugin (deploy-target@1) needs a new table, a site or API setting, a custom domain or an alarm address, or when its deploy workflow fails. The Terraform and both workflows are managed and pinned; the build changes albitor.yaml and infrastructure/aws/terraform.tfvars, never the rendered files.
metadata:
  capability: deploy-target
---

# AWS deploy target (deploy-byoa-aws)

The stack under `infrastructure/aws/` reads `albitor.yaml` itself at plan time and builds what it
declares. There is nothing to regenerate: change the manifest and the next deploy changes the
infrastructure.

## What is managed (never edit)

The workflows `ci.yml`, `deploy.yml`, `destroy.yml` and `invite.yml` under `.github/workflows/`,
`infrastructure/aws/*.tf`, `infrastructure/aws/.terraform.lock.hcl` and the three scripts under
`scripts/`. A finding in them is the plugin's, not yours: a change to how the stack builds is a new release of the plugin, offered as an
upgrade change request.

## What you may change

| To | Edit |
| --- | --- |
| Add or change a table | `services.tables` in `albitor.yaml` (up to five; a table dropped from the manifest fails the apply while `deletion_protection` is on) |
| Serve the site on a domain, or choose who hears about alarms | `infrastructure/aws/terraform.tfvars` (`dns_mode`, `domain_name`, `parent_zone_id`, `domain_ready`, `alarm_email`) |

The stack renders `site`, `http` (runtime `go`), `services.tables` and `services.auth` (a Cognito
pool, public client and groups; the function gets `AUTH_USER_POOL_ID` and `AUTH_CLIENT_ID`). It
refuses, at plan time, a manifest that declares `services.objects`, `schedules` or `config`, a runtime
other than `go`, or build outputs other than `web/build` and `api/dist`. Do not work around the
refusal: remove the block or choose another deploy target.

Who can sign in is handled for you: every `dev` deploy seeds the demo and verifier accounts
(`scripts/seed-platform-accounts.sh`, under the deploy role, passwords from the app's secret store),
and the owner invites everyone else by running the `invite` workflow (`.github/workflows/invite.yml`).
Do not seed or create users in the app, in the stack or with a user-pool admin call: the function
holds none, and the floor refuses a grant. The first sign-in of an invited person is the
`auth-cognito` plugin's `new_password_required` step. The seed reads the passwords from the path
segment Albitor wrote into `env/<environment>.deploy.env` (`ALBITOR_APP_SECRETS_SEGMENT`), never from
the repository name, and the deploy receipt reports the pool, the client and the seed's status.
When the manifest declares `services.auth.groups`, the same seed adds both accounts to
`services.auth.admin_group` (default: the first group) and to no other group, and gives every
declared group an account of its own (`<role>@albitor-role.invalid`, that group alone), so a spec
signs in as one role through `auth.signInAs(role)`. The receipt's `seeded_roles` reports the roles
seeded. Never ask a person to add accounts to groups.
Inviting needs write access to the repository; `docs/inviting-people.md` is delivered at handover.
See the README.

## Custom domains

`dns_mode = "none"` serves the CloudFront URL. `in_account` needs the parent zone in the same
account. `external_delegated` creates the child zone and certificate on the first apply and prints
`route53_nameservers`: delegate them, then set `domain_ready = true` and deploy again.

## The deploy

A push to main builds and tests, applies the Terraform under the deploy role named in
`infrastructure/aws/env/dev.deploy.env`, syncs `web/build` to the site bucket and waits for the
CloudFront invalidation. The `env/` files are written by the platform when the repository is bound
to an account; never write them. A deploy that fails at "resolve the deploy identity" means the
repository is not bound yet.
