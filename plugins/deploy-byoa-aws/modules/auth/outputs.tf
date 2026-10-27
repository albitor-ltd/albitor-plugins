output "user_pool_id" {
  description = "The user pool id."
  value       = aws_cognito_user_pool.this.id
}

output "client_id" {
  description = "The public app client id."
  value       = aws_cognito_user_pool_client.this.id
}

output "groups" {
  description = "The group names created in the pool."
  value       = sort(keys(aws_cognito_user_group.this))
}

output "admin_group" {
  description = "The resolved admin group the seed puts the platform's demo and verifier accounts into: services.auth.admin_group, else the first group, else empty (an app with no groups has no roles)."
  value       = var.admin_group != "" ? var.admin_group : (length(var.groups) > 0 ? var.groups[0] : "")
}

output "invite_only" {
  description = "Whether only an administrator can create accounts (services.auth.sign_up is invite_only)."
  value       = aws_cognito_user_pool.this.admin_create_user_config[0].allow_admin_create_user_only
}

output "env" {
  description = "The environment the function is handed: AUTH_USER_POOL_ID and AUTH_CLIENT_ID, which the auth-cognito plugin reads (Lambda sets AWS_REGION itself)."
  value = {
    AUTH_USER_POOL_ID = aws_cognito_user_pool.this.id
    AUTH_CLIENT_ID    = aws_cognito_user_pool_client.this.id
  }
}
