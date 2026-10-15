locals {
  # A custom domain is in play for both Route 53 modes.
  custom_domain = var.dns_mode != "none"

  # Whether to attach the domain to CloudFront and wait on ACM validation. Staged behind
  # domain_ready so the first apply never blocks on a delegation that is not live yet (ADR-0041).
  attach_custom_domain = local.custom_domain && var.domain_ready

  # in_account is the only mode where Terraform owns the parent zone and can write the delegation.
  write_delegation = var.dns_mode == "in_account"

  has_api = var.api_enabled
}

# --- Bundle bucket (private; reached only through CloudFront) -----------------

resource "aws_s3_bucket" "spa" {
  bucket = "${var.name_prefix}-spa"
  tags   = var.tags
}

resource "aws_s3_bucket_public_access_block" "spa" {
  bucket                  = aws_s3_bucket.spa.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "spa" {
  bucket = aws_s3_bucket.spa.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_cloudfront_origin_access_control" "spa" {
  name                              = "${var.name_prefix}-spa"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# --- CloudFront distribution ---------------------------------------------------

resource "aws_cloudfront_distribution" "this" {
  enabled             = true
  comment             = var.name_prefix
  default_root_object = var.index_document
  price_class         = "PriceClass_100"
  aliases             = local.attach_custom_domain ? [var.domain_name] : []
  tags                = var.tags

  origin {
    origin_id                = "spa"
    domain_name              = aws_s3_bucket.spa.bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.spa.id
  }

  dynamic "origin" {
    for_each = local.has_api ? [var.api_origin_host] : []
    content {
      origin_id   = "api"
      domain_name = origin.value
      custom_origin_config {
        http_port              = 80
        https_port             = 443
        origin_protocol_policy = "https-only"
        origin_ssl_protocols   = ["TLSv1.2"]
      }
    }
  }

  # The bundle is the default origin.
  default_cache_behavior {
    target_origin_id           = "spa"
    viewer_protocol_policy     = "redirect-to-https"
    allowed_methods            = ["GET", "HEAD", "OPTIONS"]
    cached_methods             = ["GET", "HEAD"]
    cache_policy_id            = "658327ea-f89d-4fab-a63d-7e88639e58f6" # Managed-CachingOptimized
    response_headers_policy_id = "67f7725c-6f97-4210-82d7-5512b31e9d03" # Managed-SecurityHeadersPolicy
    compress                   = true
  }

  # /api/* goes to API Gateway, uncached, forwarding everything except Host.
  dynamic "ordered_cache_behavior" {
    for_each = local.has_api ? [1] : []
    content {
      path_pattern             = "/api/*"
      target_origin_id         = "api"
      viewer_protocol_policy   = "redirect-to-https"
      allowed_methods          = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
      cached_methods           = ["GET", "HEAD"]
      cache_policy_id          = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad" # Managed-CachingDisabled
      origin_request_policy_id = "b689b0a8-53d0-40ab-baf2-68738e2966ac" # Managed-AllViewerExceptHostHeader
      compress                 = true
    }
  }

  # Single-page app routing: a path the bucket does not hold returns the index, not S3's 403/404.
  dynamic "custom_error_response" {
    for_each = var.spa ? [403, 404] : []
    content {
      error_code            = custom_error_response.value
      response_code         = 200
      response_page_path    = "/${var.index_document}"
      error_caching_min_ttl = 10
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  # The default CloudFront certificate until the custom domain is validated and attached (staged
  # behind domain_ready); then the us-east-1 ACM certificate takes over.
  dynamic "viewer_certificate" {
    for_each = local.attach_custom_domain ? [] : [1]
    content {
      cloudfront_default_certificate = true
    }
  }
  dynamic "viewer_certificate" {
    for_each = local.attach_custom_domain ? [1] : []
    content {
      # The validation resource, not the certificate: CloudFront only ever attaches an ISSUED one.
      acm_certificate_arn      = aws_acm_certificate_validation.cert[0].certificate_arn
      ssl_support_method       = "sni-only"
      minimum_protocol_version = "TLSv1.2_2021"
    }
  }
}

# Only this distribution may read the bucket.
data "aws_iam_policy_document" "spa" {
  statement {
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.spa.arn}/*"]
    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.this.arn]
    }
  }
}

resource "aws_s3_bucket_policy" "spa" {
  bucket = aws_s3_bucket.spa.id
  policy = data.aws_iam_policy_document.spa.json
}

# --- Custom domain (dns_mode in_account or external_delegated) ----------------

# The child hosted zone holds the validation records and the alias records. It is created for both
# modes on the first apply.
resource "aws_route53_zone" "app" {
  count = local.custom_domain ? 1 : 0
  name  = var.domain_name
  tags  = var.tags
}

# CloudFront requires its certificate in us-east-1, wherever the rest of the stack runs.
resource "aws_acm_certificate" "cert" {
  count             = local.custom_domain ? 1 : 0
  provider          = aws.us_east_1
  domain_name       = var.domain_name
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = var.tags
}

# The validation records live in the child zone this module owns, so they are written immediately
# whatever state the parent delegation is in.
resource "aws_route53_record" "cert_validation" {
  for_each = local.custom_domain ? {
    for dvo in aws_acm_certificate.cert[0].domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      type   = dvo.resource_record_type
      record = dvo.resource_record_value
    }
  } : {}

  zone_id         = aws_route53_zone.app[0].zone_id
  name            = each.value.name
  type            = each.value.type
  records         = [each.value.record]
  ttl             = 60
  allow_overwrite = true
}

# The wait runs only once the domain is ready to attach: the first apply for in_account (the
# delegation is in-account), the second for external_delegated.
resource "aws_acm_certificate_validation" "cert" {
  count                   = local.attach_custom_domain ? 1 : 0
  provider                = aws.us_east_1
  certificate_arn         = aws_acm_certificate.cert[0].arn
  validation_record_fqdns = [for r in aws_route53_record.cert_validation : r.fqdn]
}

resource "aws_route53_record" "alias_a" {
  count   = local.attach_custom_domain ? 1 : 0
  zone_id = aws_route53_zone.app[0].zone_id
  name    = var.domain_name
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.this.domain_name
    zone_id                = aws_cloudfront_distribution.this.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "alias_aaaa" {
  count   = local.attach_custom_domain ? 1 : 0
  zone_id = aws_route53_zone.app[0].zone_id
  name    = var.domain_name
  type    = "AAAA"

  alias {
    name                   = aws_cloudfront_distribution.this.domain_name
    zone_id                = aws_cloudfront_distribution.this.hosted_zone_id
    evaluate_target_health = false
  }
}

# in_account: the parent zone is in this account, so Terraform writes the NS delegation and the
# child zone resolves with no operator step. For external_delegated the operator delegates by hand
# from route53_nameservers.
resource "aws_route53_record" "delegation" {
  count   = local.write_delegation ? 1 : 0
  zone_id = var.parent_zone_id
  name    = var.domain_name
  type    = "NS"
  ttl     = 300
  records = aws_route53_zone.app[0].name_servers
}
