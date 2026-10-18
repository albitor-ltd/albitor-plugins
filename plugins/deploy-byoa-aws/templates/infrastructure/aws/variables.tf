variable "project_name" {
  description = "Short name for the app; the prefix of every resource name. The pipeline derives it from the repository name."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,30}$", var.project_name))
    error_message = "project_name must be 2 to 31 lower-case letters, digits and hyphens, starting with a letter."
  }
}

variable "environment" {
  description = "Deployment environment (dev, prod, ...). Part of resource names."
  type        = string
  default     = "dev"

  validation {
    condition     = can(regex("^[a-z][a-z0-9]{1,8}$", var.environment))
    error_message = "environment must be 2 to 9 lower-case letters and digits, starting with a letter."
  }
}

variable "aws_region" {
  description = "AWS region for the regional resources (Lambda, API Gateway, S3, DynamoDB)."
  type        = string
  default     = "eu-west-2"
}

# --- DNS / custom domain -----------------------------------------------------

variable "dns_mode" {
  description = <<-EOT
    How the app is served. No default: DNS is the biggest source of deploy pain and the one thing
    that varies per deployment, so terraform.tfvars states it (ADR-0041).
      - none               serve on the CloudFront URL; no Route 53, no ACM.
      - in_account         the parent zone is in this account: Terraform creates the child zone,
                           the certificate, its validation and the NS delegation.
      - external_delegated the parent zone is elsewhere: Terraform creates the child zone and
                           certificate and emits route53_nameservers; delegate, then apply again
                           with domain_ready = true.
  EOT
  type        = string
  nullable    = false

  validation {
    condition     = contains(["none", "in_account", "external_delegated"], var.dns_mode)
    error_message = "dns_mode must be one of: none, in_account, external_delegated."
  }
}

variable "domain_name" {
  description = "Fully qualified domain to serve the app on. Required unless dns_mode is none."
  type        = string
  default     = ""
}

variable "parent_zone_id" {
  description = "Route 53 hosted zone id of the in-account parent zone. Required for dns_mode = in_account."
  type        = string
  default     = ""
}

variable "domain_ready" {
  description = "Attach the custom domain to CloudFront. False on the first apply of external_delegated, true once the delegation is live (in_account may be true from the start)."
  type        = bool
  default     = false
}

# --- Security headers --------------------------------------------------------

variable "csp_script_hashes" {
  description = "Extra CSP script sources ('sha256-...', quoted, space-separated), added to the hashes the stack computes itself from the inline scripts in the built web shell. Usually empty."
  type        = string
  default     = ""
}

variable "csp_previous_script_hashes" {
  description = "Set by the deploy workflow, not here: the live stack's csp_script_hashes output read before the apply. The policy admits the previous build's shell beside the new one, so the site keeps working between the apply and the bundle landing."
  type        = string
  default     = ""

  validation {
    condition     = can(regex("^('sha(256|384|512)-[A-Za-z0-9+/]+=*'( +'sha(256|384|512)-[A-Za-z0-9+/]+=*')* *)?$", var.csp_previous_script_hashes))
    error_message = "csp_previous_script_hashes must be space-separated quoted hash sources such as 'sha256-abc...='."
  }
}

variable "content_security_policy" {
  description = "Replaces the whole Content-Security-Policy, script-src hashes included. Empty uses the closed default (same-origin only)."
  type        = string
  default     = ""
}

variable "hsts_include_subdomains" {
  description = "Add includeSubDomains to Strict-Transport-Security. Off by default: other hosts under the app's domain may not be ready to be HTTPS-only."
  type        = bool
  default     = false
}

variable "hsts_preload" {
  description = "Add preload to Strict-Transport-Security. Off by default: preloading is a decision for the domain's owner and slow to undo."
  type        = bool
  default     = false
}

# --- Operations --------------------------------------------------------------

variable "alarm_email" {
  description = "An address to subscribe to the alarm topic. Empty subscribes nobody."
  type        = string
  default     = ""
}

variable "deletion_protection" {
  description = "Protect each DynamoDB table and the Cognito user pool from deletion. Leave true: one dropped from albitor.yaml then fails the apply instead of destroying its data."
  type        = bool
  default     = true
}

# --- Where things are --------------------------------------------------------

variable "manifest_path" {
  description = "Path to albitor.yaml. Null reads the repository's own, two directories up."
  type        = string
  default     = null
}

variable "build_root" {
  description = "Directory the built artefacts sit under (api/dist/...), relative to this directory. The pipeline sets it to the downloaded build artefact."
  type        = string
  default     = "../.."
}

variable "web_build_dir" {
  description = "Directory holding the built web bundle, relative to this directory. Null uses web/build under build_root. The stack hashes the inline scripts in its HTML for the CSP."
  type        = string
  default     = null
}
