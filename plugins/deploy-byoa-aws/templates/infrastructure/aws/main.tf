# Managed by Albitor (deploy-byoa-aws): rendered from albitor.yaml, pinned in .albitor/lock.json.
# Edit albitor.yaml and terraform.tfvars, not this file. A change to what the stack builds is a
# change to the manifest; a change to how it builds it is a new release of the plugin.
#
# The resources live in versioned modules, fetched at the plugin release this file was rendered
# from (?ref=). The stack reads the manifest itself at plan time: nothing here is generated per
# app, so the same file renders every app and a manifest edit is the whole change.

locals {
  manifest_file = coalesce(var.manifest_path, "${path.root}/../../albitor.yaml")
  manifest      = yamldecode(file(local.manifest_file))

  site   = try(local.manifest.site, null)
  http   = try(local.manifest.http, null)
  tables = try(local.manifest.services.tables, [])
  auth   = try(local.manifest.services.auth, null)

  # The manifest names where the built artefacts are; the pipeline ships exactly these two
  # directories (it builds, then deploys, in separate jobs), so the manifest has to agree.
  static_output = try(local.manifest.build.static.output, "")
  api_output    = try(local.manifest.build.api.output, "")
  # http.handler joins the output directory here and is checked once, by the api module's `binary`
  # variable (a clean relative path, no .. segment), which is where a hostile value stops.
  api_binary = local.http == null ? "" : "${local.api_output}/${try(local.http.handler, "")}"

  # Blocks of albitor.yaml this target does not render. Planning an app that needs one would
  # deploy it without the thing it asked for, so the stack refuses instead.
  unsupported = [
    for k, v in {
      "services.objects" = try(local.manifest.services.objects, [])
      "schedules"        = try(local.manifest.schedules, [])
      "config"           = try(local.manifest.config, [])
    } : k if v != null && v != []
  ]

  name_prefix = "${var.project_name}-${var.environment}"
}

# The manifest is not trusted: it is agent-written. Everything it steers is checked here or in the
# module variable it feeds, and a failed check stops the plan.
resource "terraform_data" "manifest" {
  lifecycle {
    precondition {
      condition     = try(local.manifest.version, "") == "albitor/v1"
      error_message = "albitor.yaml must declare version: albitor/v1."
    }
    precondition {
      condition     = local.site != null || local.http != null
      error_message = "albitor.yaml declares neither site nor http: there is nothing to serve."
    }
    precondition {
      condition     = local.http == null || try(local.http.runtime, "") == "go"
      error_message = "http.runtime must be go: deploy-byoa-aws runs a Go binary behind the Lambda Web Adapter."
    }
    precondition {
      condition     = length(local.unsupported) == 0
      error_message = "albitor.yaml declares ${join(", ", local.unsupported)}, which deploy-byoa-aws does not render yet. It renders site, http, services.tables and services.auth. Remove the block, or switch to a deploy target that supports it."
    }
    precondition {
      condition     = local.site == null || local.static_output == "web/build"
      error_message = "build.static.output must be web/build, the directory the deploy workflow ships."
    }
    precondition {
      condition     = local.http == null || local.api_output == "api/dist"
      error_message = "build.api.output must be api/dist, the directory the deploy workflow ships."
    }
  }
}

module "data" {
  source = "git::https://github.com/albitor-ltd/albitor-plugins.git//plugins/deploy-byoa-aws/modules/data?ref=deploy-byoa-aws/v0.3.2"

  name_prefix         = local.name_prefix
  tables              = local.tables
  deletion_protection = var.deletion_protection
}

module "auth" {
  source = "git::https://github.com/albitor-ltd/albitor-plugins.git//plugins/deploy-byoa-aws/modules/auth?ref=deploy-byoa-aws/v0.3.2"
  count  = local.auth == null ? 0 : 1

  name_prefix         = local.name_prefix
  app_name            = var.project_name
  sign_in_url         = var.dns_mode != "none" && var.domain_name != "" && var.domain_ready ? "https://${var.domain_name}" : ""
  provider_name       = try(local.auth.provider, "")
  sign_up             = try(local.auth.sign_up, "invite_only")
  groups              = try(local.auth.groups, [])
  deletion_protection = var.deletion_protection
}

module "api" {
  source = "git::https://github.com/albitor-ltd/albitor-plugins.git//plugins/deploy-byoa-aws/modules/api?ref=deploy-byoa-aws/v0.3.2"
  count  = local.http == null ? 0 : 1

  name_prefix = local.name_prefix
  region      = var.aws_region
  build_root  = var.build_root
  binary      = local.api_binary
  environment = merge(module.data.table_env, try(module.auth[0].env, {}))
  table_arns  = module.data.table_arns
}

module "site" {
  source = "git::https://github.com/albitor-ltd/albitor-plugins.git//plugins/deploy-byoa-aws/modules/site?ref=deploy-byoa-aws/v0.3.2"
  count  = local.site == null ? 0 : 1

  providers = {
    aws           = aws
    aws.us_east_1 = aws.us_east_1
  }

  name_prefix     = local.name_prefix
  api_enabled     = local.http != null
  api_origin_host = local.http == null ? "" : module.api[0].api_host
  spa             = try(local.site.spa, true)
  index_document  = try(local.site.index, "index.html")
  dns_mode        = var.dns_mode
  domain_name     = var.domain_name
  parent_zone_id  = var.parent_zone_id
  domain_ready    = var.domain_ready

  csp_script_hashes       = var.csp_script_hashes
  content_security_policy = var.content_security_policy
}

module "alarms" {
  source = "git::https://github.com/albitor-ltd/albitor-plugins.git//plugins/deploy-byoa-aws/modules/alarms?ref=deploy-byoa-aws/v0.3.2"

  name_prefix   = local.name_prefix
  api_enabled   = local.http != null
  function_name = local.http == null ? "" : module.api[0].function_name
  api_id        = local.http == null ? "" : module.api[0].api_id
  table_names   = module.data.table_names
  alarm_email   = var.alarm_email
}
