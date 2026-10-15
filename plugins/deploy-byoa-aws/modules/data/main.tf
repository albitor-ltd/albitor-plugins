locals {
  type_codes = { string = "S", number = "N", binary = "B" }

  tables = { for t in var.tables : t.name => t }

  # Every key attribute of a table (its own keys and its indexes') once, with DynamoDB's type code.
  attributes = {
    for name, t in local.tables : name => {
      for a in concat(
        [t.partition_key],
        t.sort_key == null ? [] : [t.sort_key],
        flatten([for i in t.indexes : concat([i.partition_key], i.sort_key == null ? [] : [i.sort_key])]),
      ) : a.name => local.type_codes[a.type]
    }
  }
}

resource "aws_dynamodb_table" "this" {
  for_each = local.tables

  # Derived from the app's prefix and the logical name; the manifest never names an AWS resource.
  name         = "${var.name_prefix}-${each.key}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = each.value.partition_key.name
  range_key    = each.value.sort_key == null ? null : each.value.sort_key.name

  # A table the manifest stops declaring is a destroy in the plan. Protection makes the apply fail
  # on it instead of deleting data; lift it deliberately (deletion_protection = false) to remove one.
  deletion_protection_enabled = var.deletion_protection

  dynamic "attribute" {
    for_each = local.attributes[each.key]
    content {
      name = attribute.key
      type = attribute.value
    }
  }

  dynamic "global_secondary_index" {
    for_each = { for i in each.value.indexes : i.name => i }
    content {
      name            = global_secondary_index.key
      projection_type = upper(global_secondary_index.value.projection)

      key_schema {
        attribute_name = global_secondary_index.value.partition_key.name
        key_type       = "HASH"
      }

      dynamic "key_schema" {
        for_each = global_secondary_index.value.sort_key == null ? [] : [global_secondary_index.value.sort_key]
        content {
          attribute_name = key_schema.value.name
          key_type       = "RANGE"
        }
      }
    }
  }

  point_in_time_recovery {
    enabled = true
  }

  server_side_encryption {
    enabled = true
  }

  tags = var.tags
}
