output "app_url" {
  description = "The URL the app is served on: the custom domain once attached, else the CloudFront URL."
  value       = local.attach_custom_domain ? "https://${var.domain_name}" : "https://${aws_cloudfront_distribution.this.domain_name}"
}

output "cloudfront_url" {
  description = "The CloudFront distribution URL. It always works, even before a custom domain is attached."
  value       = "https://${aws_cloudfront_distribution.this.domain_name}"
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution id, for the cache invalidation after a deploy."
  value       = aws_cloudfront_distribution.this.id
}

output "spa_bucket" {
  description = "The bucket the built bundle is synced to."
  value       = aws_s3_bucket.spa.bucket
}

output "route53_nameservers" {
  description = "For external_delegated: delegate the domain to these name servers from its parent zone, then apply again with domain_ready = true. Empty for dns_mode none."
  value       = local.custom_domain ? aws_route53_zone.app[0].name_servers : []
}
