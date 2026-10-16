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

`.github/workflows/ci.yml`, `.github/workflows/deploy.yml`, `infrastructure/aws/*.tf`,
`infrastructure/aws/.terraform.lock.hcl` and `scripts/tf-unlock-stale.sh`. A finding in them is the
plugin's, not yours: a change to how the stack builds is a new release of the plugin, offered as an
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

With `sign_up: invite_only` (the default) the pool starts empty and nothing seeds a user, so nobody
can sign in until the operator creates one under their own AWS identity, with the email lowercased:
`aws cognito-idp admin-create-user --user-pool-id <auth_user_pool_id output> --username <email>
--user-attributes Name=email,Value=<email> Name=email_verified,Value=true --message-action SUPPRESS`,
then `aws cognito-idp admin-set-user-password ... --password <12+ chars> --permanent` (a user left in
`FORCE_CHANGE_PASSWORD` cannot sign in through the app). See the README.

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
