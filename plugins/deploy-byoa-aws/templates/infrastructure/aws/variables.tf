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
