variable "name_prefix" {
  description = "Prefix for every AWS name the module derives, such as plantrain-dev."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,40}$", var.name_prefix))
    error_message = "name_prefix must be 2 to 41 lower-case letters, digits and hyphens, starting with a letter."
  }
}

variable "api_enabled" {
  description = "Whether the app has an http block. A literal known at plan, so CloudFront's API origin can be planned before the gateway exists."
  type        = bool
  default     = false
}

variable "api_origin_host" {
  description = "Host name of the API Gateway endpoint CloudFront routes /api/* to. Used only when api_enabled."
  type        = string
  default     = ""

  validation {
    condition     = var.api_origin_host == "" || can(regex("^[a-z0-9]+\\.execute-api\\.[a-z0-9-]+\\.amazonaws\\.com$", var.api_origin_host))
    error_message = "api_origin_host must be an API Gateway execute-api host name."
  }
}

variable "spa" {
  description = "site.spa: serve the index document for application routes (a viewer-request function on the site behaviour) so client-side routing resolves. Paths under /api/ and paths whose last segment has a dot are never rewritten."
  type        = bool
  default     = true
}

variable "index_document" {
  description = "site.index: the index document, relative to the bundle."
  type        = string
  default     = "index.html"

  validation {
    condition = (
      can(regex("^[A-Za-z0-9_][A-Za-z0-9_./-]*$", var.index_document)) &&
      !contains(split("/", var.index_document), "..") &&
      !contains(split("/", var.index_document), "")
    )
    error_message = "index_document must be a clean relative path of letters, digits and _ . / - with no .. segment."
  }
}

# --- Security headers --------------------------------------------------------

variable "csp_script_hashes" {
  description = "Space-separated CSP script sources ('sha256-...' with quotes) for the inline scripts in the built shell. A framework build inlines a bootstrap script; without its hash the browser blocks it and the page is blank. The managed root computes these at plan time from the HTML of the build about to be shipped."
  type        = string
  default     = ""

  validation {
    condition     = can(regex("^('sha(256|384|512)-[A-Za-z0-9+/]+=*'( +'sha(256|384|512)-[A-Za-z0-9+/]+=*')* *)?$", var.csp_script_hashes))
    error_message = "csp_script_hashes must be space-separated quoted hash sources such as 'sha256-abc...='."
  }
}

variable "csp_previous_script_hashes" {
  description = "The script hashes of the shell being served now (the root hashes a copy taken from the bucket before the apply), in the same form, admitted beside csp_script_hashes so it keeps working until the new bundle lands. Dropped from the policy when the union would pass CloudFront's header limit."
  type        = string
  default     = ""

  validation {
    condition     = can(regex("^('sha(256|384|512)-[A-Za-z0-9+/]+=*'( +'sha(256|384|512)-[A-Za-z0-9+/]+=*')* *)?$", var.csp_previous_script_hashes))
    error_message = "csp_previous_script_hashes must be space-separated quoted hash sources such as 'sha256-abc...='."
  }
}

variable "content_security_policy" {
  description = "Replaces the whole Content-Security-Policy, script-src included, so an override carries its own hashes. Empty uses the module's closed default."
  type        = string
  default     = ""
}

variable "hsts_max_age_seconds" {
  description = "Strict-Transport-Security max-age."
  type        = number
  default     = 31536000
}

variable "hsts_include_subdomains" {
  description = "Add includeSubDomains to Strict-Transport-Security. Off by default: the app's domain may sit under a parent whose other hosts are not this stack's to commit to HTTPS."
  type        = bool
  default     = false
}

variable "hsts_preload" {
  description = "Add preload to Strict-Transport-Security. Off by default: preload asks browsers to hard-code HTTPS for the whole registrable domain and is slow to undo, so it is the domain owner's decision."
  type        = bool
  default     = false
}

# --- DNS / custom domain -----------------------------------------------------

variable "dns_mode" {
  description = <<-EOT
    How the app is served. No default: DNS is the one thing that varies per deployment and the
    biggest source of deploy pain, so the person chooses.
      - none               serve on the CloudFront URL; no Route 53, no ACM.
      - in_account         the parent zone is in this account: the child zone, the certificate, its
                           validation and the NS delegation are all created, with no operator step.
      - external_delegated the parent zone is elsewhere: the child zone and certificate are created
                           and route53_nameservers is emitted; delegate, then apply again with
                           domain_ready = true. The first apply never waits on validation.
  EOT
  type        = string

  validation {
    condition     = contains(["none", "in_account", "external_delegated"], var.dns_mode)
    error_message = "dns_mode must be one of: none, in_account, external_delegated."
  }
}

variable "domain_name" {
  description = "Fully qualified domain to serve the app on. Required unless dns_mode is none."
  type        = string
  default     = ""

  validation {
    condition     = var.domain_name == "" || can(regex("^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z]{2,63}$", var.domain_name))
    error_message = "domain_name must be a lower-case host name such as app.example.com."
  }
  validation {
    condition     = var.dns_mode == "none" || var.domain_name != ""
    error_message = "domain_name is required when dns_mode is in_account or external_delegated."
  }
}

variable "parent_zone_id" {
  description = "Route 53 hosted zone id of the in-account parent zone. Required for dns_mode = in_account, where Terraform writes the NS delegation."
  type        = string
  default     = ""

  validation {
    condition     = var.parent_zone_id == "" || can(regex("^Z[A-Z0-9]{5,31}$", var.parent_zone_id))
    error_message = "parent_zone_id must be a Route 53 hosted zone id such as Z0123456789ABCDEFGHIJ."
  }
  validation {
    condition     = var.dns_mode != "in_account" || var.parent_zone_id != ""
    error_message = "parent_zone_id is required when dns_mode is in_account."
  }
}

variable "domain_ready" {
  description = <<-EOT
    Staging gate for attaching the custom domain. False: the apply creates the zone, the
    certificate and its validation records but neither waits for validation nor attaches the alias,
    so it cannot hang on a delegation that is not live yet. in_account can set it true from the
    start; external_delegated sets it true on the second apply, once delegation is live.
  EOT
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags for every resource."
  type        = map(string)
  default     = {}
}
