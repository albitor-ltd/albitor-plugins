variable "name_prefix" {
  description = "Prefix for every AWS name the module derives, such as plantrain-dev."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,40}$", var.name_prefix))
    error_message = "name_prefix must be 2 to 41 lower-case letters, digits and hyphens, starting with a letter."
  }
}

variable "tables" {
  description = "The albitor.yaml services.tables entries. The manifest is not trusted: every field is checked here."
  type = list(object({
    name          = string
    partition_key = object({ name = string, type = string })
    sort_key      = optional(object({ name = string, type = string }))
    indexes = optional(list(object({
      name          = string
      partition_key = object({ name = string, type = string })
      sort_key      = optional(object({ name = string, type = string }))
      projection    = optional(string, "all")
    })), [])
  }))
  default = []

  validation {
    condition     = length(var.tables) <= 5
    error_message = "At most five tables."
  }
  validation {
    condition     = length(distinct([for t in var.tables : t.name])) == length(var.tables)
    error_message = "Table names must be unique."
  }
  validation {
    condition = alltrue([
      for t in var.tables : can(regex("^[a-z][a-z0-9-]{1,31}$", t.name)) &&
      length(distinct([for i in t.indexes : i.name])) == length(t.indexes) &&
      alltrue([for i in t.indexes : can(regex("^[a-z][a-z0-9-]{1,31}$", i.name))])
    ])
    error_message = "Table and index names must match ^[a-z][a-z0-9-]{1,31}$, and an index name must be unique within its table."
  }
  validation {
    condition = alltrue([
      for t in var.tables : alltrue([
        for k in concat(
          [t.partition_key],
          t.sort_key == null ? [] : [t.sort_key],
          flatten([for i in t.indexes : concat([i.partition_key], i.sort_key == null ? [] : [i.sort_key])]),
        ) : can(regex("^[A-Za-z_][A-Za-z0-9_.-]{0,63}$", k.name)) && contains(["string", "number", "binary"], k.type)
      ])
    ])
    error_message = "A key name must match ^[A-Za-z_][A-Za-z0-9_.-]{0,63}$ and a key type must be string, number or binary."
  }
  validation {
    condition = alltrue([
      for t in var.tables : (
        length(distinct([
          for k in concat(
            [t.partition_key],
            t.sort_key == null ? [] : [t.sort_key],
            flatten([for i in t.indexes : concat([i.partition_key], i.sort_key == null ? [] : [i.sort_key])]),
          ) : "${k.name}:${k.type}"
          ])) == length(distinct([
          for k in concat(
            [t.partition_key],
            t.sort_key == null ? [] : [t.sort_key],
            flatten([for i in t.indexes : concat([i.partition_key], i.sort_key == null ? [] : [i.sort_key])]),
          ) : k.name
        ]))
      )
    ])
    error_message = "One attribute cannot be given two types within a table."
  }
  validation {
    condition     = alltrue([for t in var.tables : length(t.indexes) <= 2 && alltrue([for i in t.indexes : contains(["all", "keys_only"], i.projection)])])
    error_message = "At most two indexes per table, each projecting all or keys_only."
  }
}

variable "deletion_protection" {
  description = "Protect each table from deletion. A table dropped from the manifest then fails the apply instead of destroying its data."
  type        = bool
  default     = true
}

variable "tags" {
  description = "Tags for every resource."
  type        = map(string)
  default     = {}
}
