# Offline: the AWS provider is mocked, so these run without credentials. They assert what the
# module plans for the tables a manifest declares.
mock_provider "aws" {}

variables {
  name_prefix = "plantrain-dev"
  tables = [{
    name          = "data"
    partition_key = { name = "pk", type = "string" }
    sort_key      = { name = "sk", type = "string" }
  }]
}

run "one_table_keeps_its_keys_and_is_protected" {
  command = plan

  assert {
    condition     = aws_dynamodb_table.this["data"].name == "plantrain-dev-data"
    error_message = "the AWS name is derived from the prefix and the logical name, never taken from the manifest"
  }
  assert {
    condition     = aws_dynamodb_table.this["data"].hash_key == "pk" && aws_dynamodb_table.this["data"].range_key == "sk"
    error_message = "the table keeps the manifest's keys"
  }
  assert {
    condition     = aws_dynamodb_table.this["data"].billing_mode == "PAY_PER_REQUEST"
    error_message = "on-demand billing"
  }
  assert {
    condition     = aws_dynamodb_table.this["data"].deletion_protection_enabled
    error_message = "a table is protected from deletion by default: a manifest edit must not destroy data"
  }
  assert {
    condition     = one(aws_dynamodb_table.this["data"].point_in_time_recovery).enabled
    error_message = "point-in-time recovery is on"
  }
  assert {
    condition     = one(aws_dynamodb_table.this["data"].server_side_encryption).enabled
    error_message = "server-side encryption is on"
  }
  assert {
    condition     = output.table_env["ALBITOR_TABLE_DATA"] == "plantrain-dev-data"
    error_message = "the function is handed the table name under the variable the store reads"
  }
}

run "protection_can_be_lifted_for_a_throwaway_account" {
  command = plan
  variables {
    deletion_protection = false
  }

  assert {
    condition     = !aws_dynamodb_table.this["data"].deletion_protection_enabled
    error_message = "deletion_protection = false lifts it"
  }
}

run "key_types_and_indexes_map_to_DynamoDB_s" {
  command = plan
  variables {
    tables = [{
      name          = "order-items"
      partition_key = { name = "pk", type = "string" }
      sort_key      = { name = "seq", type = "number" }
      indexes = [{
        name          = "by-blob"
        partition_key = { name = "blob", type = "binary" }
        projection    = "keys_only"
      }]
    }]
  }

  assert {
    condition = alltrue([
      for a in aws_dynamodb_table.this["order-items"].attribute :
      a.type == { pk = "S", seq = "N", blob = "B" }[a.name]
    ]) && length(aws_dynamodb_table.this["order-items"].attribute) == 3
    error_message = "attributes are S, N and B once each"
  }
  assert {
    condition     = one(aws_dynamodb_table.this["order-items"].global_secondary_index).projection_type == "KEYS_ONLY"
    error_message = "keys_only is KEYS_ONLY"
  }
  assert {
    condition     = output.table_env["ALBITOR_TABLE_ORDER_ITEMS"] == "plantrain-dev-order-items"
    error_message = "a hyphen in the logical name becomes an underscore in the variable"
  }
}

run "an_index_projects_everything_by_default" {
  command = plan
  variables {
    tables = [{
      name          = "data"
      partition_key = { name = "pk", type = "string" }
      indexes = [{
        name          = "by-status"
        partition_key = { name = "status", type = "string" }
        sort_key      = { name = "created_at", type = "string" }
      }]
    }]
  }

  assert {
    condition     = one(aws_dynamodb_table.this["data"].global_secondary_index).projection_type == "ALL"
    error_message = "the default projection is all"
  }
  assert {
    condition     = [for k in one(aws_dynamodb_table.this["data"].global_secondary_index).key_schema : "${k.attribute_name}:${k.key_type}"] == ["status:HASH", "created_at:RANGE"]
    error_message = "an index keeps its partition and sort keys, in that order"
  }
}

run "no_tables_plans_nothing" {
  command = plan
  variables {
    tables = []
  }

  assert {
    condition     = length(aws_dynamodb_table.this) == 0 && length(output.table_arns) == 0
    error_message = "an app with no tables gets none"
  }
}

run "a_name_that_is_not_a_logical_name_is_refused" {
  command = plan
  variables {
    tables = [{
      name          = "Data;drop"
      partition_key = { name = "pk", type = "string" }
    }]
  }
  expect_failures = [var.tables]
}

run "an_unknown_key_type_is_refused" {
  command = plan
  variables {
    tables = [{
      name          = "data"
      partition_key = { name = "pk", type = "map" }
    }]
  }
  expect_failures = [var.tables]
}

run "a_key_name_that_is_not_a_plain_attribute_name_is_refused" {
  command = plan
  variables {
    tables = [{
      name          = "data"
      partition_key = { name = "pk\"; x", type = "string" }
    }]
  }
  expect_failures = [var.tables]
}

run "one_attribute_with_two_types_is_refused" {
  command = plan
  variables {
    tables = [{
      name          = "data"
      partition_key = { name = "pk", type = "string" }
      indexes = [{
        name          = "by-pk"
        partition_key = { name = "pk", type = "number" }
      }]
    }]
  }
  expect_failures = [var.tables]
}

run "a_duplicate_table_name_is_refused" {
  command = plan
  variables {
    tables = [
      { name = "data", partition_key = { name = "pk", type = "string" } },
      { name = "data", partition_key = { name = "pk", type = "string" } },
    ]
  }
  expect_failures = [var.tables]
}

run "more_than_five_tables_is_refused" {
  command = plan
  variables {
    tables = [for i in range(6) : {
      name          = "t${i}-x"
      partition_key = { name = "pk", type = "string" }
    }]
  }
  expect_failures = [var.tables]
}

run "more_than_two_indexes_on_a_table_is_refused" {
  command = plan
  variables {
    tables = [{
      name          = "data"
      partition_key = { name = "pk", type = "string" }
      indexes = [for i in range(3) : {
        name          = "idx-${i}x"
        partition_key = { name = "a${i}", type = "string" }
      }]
    }]
  }
  expect_failures = [var.tables]
}
