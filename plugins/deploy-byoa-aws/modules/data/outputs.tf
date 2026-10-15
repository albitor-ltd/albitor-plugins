output "table_names" {
  description = "AWS table name by logical name."
  value       = { for k, t in aws_dynamodb_table.this : k => t.name }
}

output "table_arns" {
  description = "ARN of each table, for the function's role."
  value       = [for t in aws_dynamodb_table.this : t.arn]
}

output "table_env" {
  description = "The environment the function is handed: ALBITOR_TABLE_<LOGICAL_NAME> is the AWS table name, the variable the data-dynamodb store reads."
  value       = { for k, t in aws_dynamodb_table.this : "ALBITOR_TABLE_${upper(replace(k, "-", "_"))}" => t.name }
}
