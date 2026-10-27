output "app_url" {
  description = "The URL the app is served on: the custom domain once attached, else the CloudFront URL, else (no site) the API endpoint."
  value       = try(module.site[0].app_url, try(module.api[0].api_endpoint, ""))
}

output "cloudfront_url" {
  description = "The CloudFront URL. It always works, even before a custom domain is attached. Empty when the app has no site."
  value       = try(module.site[0].cloudfront_url, "")
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution id, for the cache invalidation after a deploy."
  value       = try(module.site[0].cloudfront_distribution_id, "")
}

output "spa_bucket" {
  description = "The bucket the built bundle is synced to."
  value       = try(module.site[0].spa_bucket, "")
}

output "api_endpoint" {
  description = "The API Gateway endpoint (CloudFront fronts it at /api/* when there is a site)."
  value       = try(module.api[0].api_endpoint, "")
}

output "route53_nameservers" {
  description = "For external_delegated: delegate the domain to these name servers, then apply again with domain_ready = true."
  value       = try(module.site[0].route53_nameservers, [])
}

output "alarm_topic_arn" {
  description = "The SNS topic every alarm notifies. Subscribe a chat webhook or a pager to it."
  value       = module.alarms.topic_arn
}

output "auth_user_pool_id" {
  description = "The Cognito user pool id when the manifest declares services.auth, else empty. The function is handed it as AUTH_USER_POOL_ID."
  value       = try(module.auth[0].user_pool_id, "")
}

output "auth_client_id" {
  description = "The Cognito public app client id when the manifest declares services.auth, else empty. The function is handed it as AUTH_CLIENT_ID."
  value       = try(module.auth[0].client_id, "")
}

output "auth_admin_group" {
  description = "The resolved services.auth.admin_group (the first group when the manifest names none, empty when it declares no groups). The deploy's seed adds the demo and verifier accounts to it."
  value       = try(module.auth[0].admin_group, "")
}

output "csp_script_hashes" {
  description = "The script hashes of this build's inline scripts (empty with no site): what the policy admits for the shell being shipped."
  value       = join(" ", local.script_hashes.current)
}
