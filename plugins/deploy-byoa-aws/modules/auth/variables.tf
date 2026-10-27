variable "name_prefix" {
  description = "Prefix for every AWS name the module derives, such as plantrain-dev."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,40}$", var.name_prefix))
    error_message = "name_prefix must be 2 to 41 lower-case letters, digits and hyphens, starting with a letter."
  }
}

variable "app_name" {
  description = "The app's name as the invitation email says it: the project name, since the manifest carries none."
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9][A-Za-z0-9 ._-]{0,63}$", var.app_name))
    error_message = "app_name must be 1 to 64 letters, digits, spaces, dots, underscores and hyphens, starting with a letter or digit."
  }
}

variable "sign_in_url" {
  description = "Where the app signs people in, quoted in the invitation. Empty when the stack cannot know it before the apply (the CloudFront address is created after the pool); the email then leaves the link out."
  type        = string
  default     = ""

  validation {
    condition     = var.sign_in_url == "" || can(regex("^https://[a-z0-9.-]{1,253}(/[A-Za-z0-9._~/-]*)?$", var.sign_in_url))
    error_message = "sign_in_url must be empty or a plain https URL."
  }
}

variable "provider_name" {
  description = "services.auth.provider from albitor.yaml. Only cognito is rendered. The manifest is not trusted."
  type        = string

  validation {
    condition     = var.provider_name == "cognito"
    error_message = "services.auth.provider must be cognito."
  }
}

variable "sign_up" {
  description = "services.auth.sign_up: invite_only (an administrator creates every account) or open (anyone may register, confirmed by email)."
  type        = string
  default     = "invite_only"

  validation {
    condition     = contains(["invite_only", "open"], var.sign_up)
    error_message = "services.auth.sign_up must be invite_only or open."
  }
}

variable "groups" {
  description = "services.auth.groups: group names to create in the pool."
  type        = list(string)
  default     = []

  validation {
    condition     = alltrue([for g in var.groups : can(regex("^[a-z][a-z0-9-]{1,31}$", g))])
    error_message = "Each group name must match ^[a-z][a-z0-9-]{1,31}$."
  }
  validation {
    condition     = length(distinct(var.groups)) == length(var.groups)
    error_message = "Group names must be unique."
  }
  validation {
    condition     = length(var.groups) <= 20
    error_message = "At most twenty groups."
  }
}

variable "admin_group" {
  description = "services.auth.admin_group: the group the platform seeds the demo and verifier accounts into. Empty means groups[0], or none when there are no groups."
  type        = string
  default     = ""

  validation {
    condition     = var.admin_group == "" || contains(var.groups, var.admin_group)
    error_message = "admin_group must be one of groups."
  }
}

variable "deletion_protection" {
  description = "Refuse to delete the pool (and every account in it). Lift it deliberately to remove the pool."
  type        = bool
  default     = true
}

variable "tags" {
  description = "Tags for every resource the module creates."
  type        = map(string)
  default     = {}
}
