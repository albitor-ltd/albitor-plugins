locals {
  admin_group = var.admin_group != "" ? var.admin_group : (length(var.groups) > 0 ? var.groups[0] : "")
}

output "user_pool_id" {
  description = "The user pool id."
  value       = aws_cognito_user_pool.this.id
}

output "user_pool_arn" {
  description = "The user pool ARN: the one resource the API's user-administration policy may act on."
  value       = aws_cognito_user_pool.this.arn
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
  value       = local.admin_group
}

output "user_admin_group" {
  description = "The group whose members administer people in an invite_only app (the seed puts the owner in it); empty in open mode, where nothing administers people."
  value       = var.sign_up == "invite_only" ? var.user_admin_group : ""
}

output "invite_only" {
  description = "Whether only an administrator can create accounts (services.auth.sign_up is invite_only)."
  value       = var.sign_up == "invite_only"
}

output "env" {
  description = "The environment the function is handed: AUTH_USER_POOL_ID, AUTH_CLIENT_ID, AUTH_SIGN_UP (invite_only or open: which API actions answer) and AUTH_ADMIN_GROUP (the app's admin role) and AUTH_USER_ADMIN_GROUP (who may administer people; empty in open), which the auth-cognito plugin reads (Lambda sets AWS_REGION itself)."
  value = {
    AUTH_USER_POOL_ID     = aws_cognito_user_pool.this.id
    AUTH_CLIENT_ID        = aws_cognito_user_pool_client.this.id
    AUTH_SIGN_UP          = var.sign_up
    AUTH_ADMIN_GROUP      = local.admin_group
    AUTH_USER_ADMIN_GROUP = var.sign_up == "invite_only" ? var.user_admin_group : ""
  }
}
