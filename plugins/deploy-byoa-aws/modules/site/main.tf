locals {
  # A custom domain is in play for both Route 53 modes.
  custom_domain = var.dns_mode != "none"

  # Whether to attach the domain to CloudFront and wait on ACM validation. Staged behind
  # domain_ready so the first apply never blocks on a delegation that is not live yet (ADR-0041).
  attach_custom_domain = local.custom_domain && var.domain_ready

  # in_account is the only mode where Terraform owns the parent zone and can write the delegation.
  write_delegation = var.dns_mode == "in_account"

  has_api = var.api_enabled

  # The default policy is closed: same-origin everything, no framing, scripts only from the bundle
  # plus the inline bootstrap a framework build puts in its shell, named by hash rather than by
  # 'unsafe-inline'. style-src-attr alone admits style="" attributes (headless UI libraries write
  # them to lock scrolling behind a dialog); <style> elements and stylesheets stay on 'self'. The
  # browser calls only its own origin: the API is reached through /api/* on this distribution.
  #
  # A deploy changes the policy before it ships the bundle (the apply waits for the distribution,
  # then the workflow syncs S3 and invalidates), and a framework's inline bootstrap names its
  # content-hashed chunks, so its hash changes every build. The previous build's hashes are
  # therefore admitted beside the new ones for that one deploy: the shell already live, or cached,
  # keeps running until the new one replaces it. They are best effort: when the union would pass
  # CloudFront's limit on the header, the policy carries the current build's alone.
  csp_current_hashes  = compact(split(" ", trimspace(var.csp_script_hashes)))
  csp_previous_hashes = compact(split(" ", trimspace(var.csp_previous_script_hashes)))
  csp_union_hashes    = distinct(concat(local.csp_current_hashes, local.csp_previous_hashes))
  csp_other_directives = [
    "style-src 'self'",
    "style-src-attr 'unsafe-inline'",
    "img-src 'self' data:",
    "font-src 'self'",
    "connect-src 'self'",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
  ]
  csp_with_previous = join("; ", concat(["default-src 'self'", join(" ", concat(["script-src 'self'"], local.csp_union_hashes))], local.csp_other_directives))
  csp_current_only  = join("; ", concat(["default-src 'self'", join(" ", concat(["script-src 'self'"], local.csp_current_hashes))], local.csp_other_directives))
  csp_default       = length(local.csp_with_previous) <= local.csp_max_length ? local.csp_with_previous : local.csp_current_only

  content_security_policy = var.content_security_policy != "" ? var.content_security_policy : local.csp_default

  # CloudFront's quota on a response headers policy's Content-Security-Policy value.
  csp_max_length = 1783
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

# --- Security headers ----------------------------------------------------------

# Attached to BOTH behaviours: a policy on the site alone would leave every API response unheadered.
resource "aws_cloudfront_response_headers_policy" "security" {
  name    = "${var.name_prefix}-security-headers"
  comment = "HSTS, CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy"

  lifecycle {
    precondition {
      condition     = length(local.content_security_policy) <= local.csp_max_length
      error_message = "The Content-Security-Policy is ${length(local.content_security_policy)} characters; CloudFront allows ${local.csp_max_length}. The built shell has too many distinct inline scripts to admit by hash: inline fewer (one bootstrap per page), or set content_security_policy."
    }
  }

  security_headers_config {
    strict_transport_security {
      access_control_max_age_sec = var.hsts_max_age_seconds
      include_subdomains         = var.hsts_include_subdomains
      preload                    = var.hsts_preload
      override                   = true
    }
    content_security_policy {
      content_security_policy = local.content_security_policy
      override                = true
    }
    frame_options {
      frame_option = "DENY"
      override     = true
    }
    content_type_options {
      override = true
    }
    referrer_policy {
      referrer_policy = "strict-origin-when-cross-origin"
      override        = true
    }
  }
}

# --- Single-page app routing at the edge ---------------------------------------

# A function on the default behaviour, not a custom_error_response: that one is distribution-wide
# and would return the shell for /api/<unknown> and for a missing asset. See functions/spa-router.js.
resource "aws_cloudfront_function" "spa_router" {
  count   = var.spa ? 1 : 0
  name    = "${var.name_prefix}-spa-router"
  runtime = "cloudfront-js-2.0"
  comment = "Rewrite application routes to the index document; never touches /api/ or files"
  publish = true
  code    = templatefile("${path.module}/functions/spa-router.js", { index_path = var.index_document })
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
    response_headers_policy_id = aws_cloudfront_response_headers_policy.security.id
    compress                   = true

    dynamic "function_association" {
      for_each = var.spa ? [1] : []
      content {
        event_type   = "viewer-request"
        function_arn = aws_cloudfront_function.spa_router[0].arn
      }
    }
  }

  # /api/* goes to API Gateway, uncached, forwarding everything except Host.
  dynamic "ordered_cache_behavior" {
    for_each = local.has_api ? [1] : []
    content {
      path_pattern               = "/api/*"
      target_origin_id           = "api"
      viewer_protocol_policy     = "redirect-to-https"
      allowed_methods            = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
      cached_methods             = ["GET", "HEAD"]
      cache_policy_id            = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad" # Managed-CachingDisabled
      origin_request_policy_id   = "b689b0a8-53d0-40ab-baf2-68738e2966ac" # Managed-AllViewerExceptHostHeader
      response_headers_policy_id = aws_cloudfront_response_headers_policy.security.id
      compress                   = true
    }
  }

  # Deliberately no custom_error_response: it is distribution-wide, so it would answer the shell
  # for /api/<unknown> and for a missing asset. A missing file surfaces S3's own 403 or 404.

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
