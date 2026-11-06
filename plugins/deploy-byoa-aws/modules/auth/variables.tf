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

variable "signup_domains" {
  description = "services.auth.signup_domains: email domains allowed to register, `example.org` or `*.example.org`. Empty means anyone. The API enforces it (the pool has no Lambda trigger); it never affects an existing account or an invitation."
  type        = list(string)
  default     = []

  validation {
    condition     = alltrue([for d in var.signup_domains : can(regex("^(\\*\\.)?([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z]{2,63}$", d))])
    error_message = "Each services.auth.signup_domains entry must be a lower-case domain such as example.org or *.example.org."
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

variable "user_admin_group" {
  description = "The group whose members administer people (the People screen) in an invite_only app. Created only in invite_only, where the owner is seeded into it; open mode has no people administration and creates nothing."
  type        = string
  default     = "user-admins"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,31}$", var.user_admin_group))
    error_message = "user_admin_group must match ^[a-z][a-z0-9-]{1,31}$."
  }
  validation {
    condition     = !contains(var.groups, var.user_admin_group)
    error_message = "user_admin_group must not also be one of groups: it is the module's own group for people administration."
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
