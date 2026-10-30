output "function_name" {
  description = "The API function's name."
  value       = aws_lambda_function.api.function_name
}

output "api_id" {
  description = "The HTTP API's id."
  value       = aws_apigatewayv2_api.api.id
}

output "api_endpoint" {
  description = "The API Gateway endpoint (CloudFront fronts it at /api/*)."
  value       = aws_apigatewayv2_api.api.api_endpoint
}

output "api_host" {
  description = "The API Gateway host name, the CloudFront origin for /api/*."
  value       = "${aws_apigatewayv2_api.api.id}.execute-api.${var.region}.amazonaws.com"
}

output "role_name" {
  description = "The function's execution role name."
  value       = aws_iam_role.lambda.name
}

output "role_arn" {
  description = "The function's execution role ARN."
  value       = aws_iam_role.lambda.arn
}

output "secrets_parameter_prefix" {
  description = "The SSM path prefix the function may read its secrets under; empty when it has no grant."
  value       = var.secrets_parameter_prefix
}
