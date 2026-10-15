variable "name_prefix" {
  description = "Prefix for every AWS name the module derives, such as plantrain-dev."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,40}$", var.name_prefix))
    error_message = "name_prefix must be 2 to 41 lower-case letters, digits and hyphens, starting with a letter."
  }
}

variable "api_enabled" {
  description = "Whether the app has an http block. A literal known at plan, so the alarms can be planned before the function exists."
  type        = bool
  default     = false
}

variable "function_name" {
  description = "The API function to alarm on. Used only when api_enabled."
  type        = string
  default     = ""
}

variable "api_id" {
  description = "The HTTP API to alarm on. Used only when api_enabled."
  type        = string
  default     = ""
}

variable "table_names" {
  description = "AWS table name by logical name; each is watched for throttling."
  type        = map(string)
  default     = {}
}

variable "alarm_email" {
  description = "An address to subscribe to the alarm topic. Empty subscribes nobody; AWS sends a confirmation the address must accept."
  type        = string
  default     = ""

  validation {
    condition     = var.alarm_email == "" || can(regex("^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$", var.alarm_email))
    error_message = "alarm_email must be one email address."
  }
}

variable "tags" {
  description = "Tags for every resource."
  type        = map(string)
  default     = {}
}
