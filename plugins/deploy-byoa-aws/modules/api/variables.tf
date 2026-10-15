variable "name_prefix" {
  description = "Prefix for every AWS name the module derives, such as plantrain-dev."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,40}$", var.name_prefix))
    error_message = "name_prefix must be 2 to 41 lower-case letters, digits and hyphens, starting with a letter."
  }
}

variable "region" {
  description = "The AWS region the function runs in; it selects the Lambda Web Adapter layer."
  type        = string

  validation {
    condition     = can(regex("^[a-z]{2}(-[a-z]+)+-[0-9]$", var.region))
    error_message = "region must look like eu-west-2."
  }
}

variable "build_root" {
  description = "Directory the built artefacts sit under, relative to the root module. Set by the pipeline, never by the manifest."
  type        = string
  default     = "../.."

  validation {
    condition     = can(regex("^[A-Za-z0-9_./-]+$", var.build_root))
    error_message = "build_root may hold letters, digits and _ . / - only."
  }
}

variable "binary" {
  description = "The built Go binary, relative to build_root: build.api.output joined to http.handler. It comes from the manifest, which is not trusted."
  type        = string

  validation {
    condition = (
      can(regex("^[A-Za-z0-9_][A-Za-z0-9_./-]*$", var.binary)) &&
      !contains(split("/", var.binary), "..") &&
      !contains(split("/", var.binary), ".") &&
      !contains(split("/", var.binary), "") &&
      length(var.binary) <= 200
    )
    error_message = "binary must be a clean relative path of letters, digits and _ . / - with no .. or . segment."
  }
}

variable "environment" {
  description = "Environment variables for the function. AWS_* and PORT belong to Lambda and the adapter and are refused."
  type        = map(string)
  default     = {}

  validation {
    condition = alltrue([
      for k, v in var.environment : can(regex("^[A-Z][A-Z0-9_]{0,63}$", k)) && !startswith(k, "AWS_") && k != "PORT"
    ])
    error_message = "Environment names must match ^[A-Z][A-Z0-9_]{0,63}$ and must not start AWS_ or be PORT: Lambda and the adapter set those."
  }
}

variable "table_arns" {
  description = "ARNs of the tables the function may use."
  type        = list(string)
  default     = []

  validation {
    condition     = alltrue([for a in var.table_arns : can(regex("^arn:aws:dynamodb:[a-z0-9-]+:[0-9]{12}:table/[A-Za-z0-9_.-]+$", a))])
    error_message = "Each table ARN must be a DynamoDB table ARN."
  }
}

variable "lambda_adapter_layer_arn" {
  description = "ARN of the AWS Lambda Web Adapter layer. Null derives the pinned x86_64 layer for the region."
  type        = string
  default     = null

  validation {
    condition     = var.lambda_adapter_layer_arn == null || can(regex("^arn:aws:lambda:[a-z0-9-]+:[0-9]{12}:layer:[A-Za-z0-9_-]+:[0-9]+$", var.lambda_adapter_layer_arn))
    error_message = "lambda_adapter_layer_arn must be a Lambda layer version ARN."
  }
}

variable "lambda_memory_mb" {
  description = "Lambda memory size in MB."
  type        = number
  default     = 256

  validation {
    condition     = var.lambda_memory_mb >= 128 && var.lambda_memory_mb <= 3008
    error_message = "lambda_memory_mb must be between 128 and 3008."
  }
}

variable "lambda_timeout_s" {
  description = "Lambda timeout in seconds. API Gateway ends a request at 30 seconds, so a longer timeout buys nothing."
  type        = number
  default     = 15

  validation {
    condition     = var.lambda_timeout_s >= 1 && var.lambda_timeout_s <= 30
    error_message = "lambda_timeout_s must be between 1 and 30."
  }
}

variable "throttle_rate" {
  description = "Steady requests per second the stage allows."
  type        = number
  default     = 50
}

variable "throttle_burst" {
  description = "Burst requests the stage allows."
  type        = number
  default     = 100
}

variable "log_retention_days" {
  description = "Days the function's logs are kept."
  type        = number
  default     = 14
}

variable "tags" {
  description = "Tags for every resource."
  type        = map(string)
  default     = {}
}
