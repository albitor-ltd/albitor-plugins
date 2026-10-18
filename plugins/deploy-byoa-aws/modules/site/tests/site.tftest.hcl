mock_provider "aws" {
  mock_data "aws_iam_policy_document" {
    defaults = {
      json = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
    }
  }
}

mock_provider "aws" {
  alias = "us_east_1"
}

# Values AWS assigns at apply, given here so a plan can be asserted on.
override_resource {
  target          = aws_s3_bucket.spa
  override_during = plan
  values = {
    id                          = "plantrain-dev-spa"
    arn                         = "arn:aws:s3:::plantrain-dev-spa"
    bucket_regional_domain_name = "plantrain-dev-spa.s3.eu-west-2.amazonaws.com"
  }
}

override_resource {
  target          = aws_cloudfront_response_headers_policy.security
  override_during = plan
  values = {
    id = "hdrpolicy-security"
  }
}

override_resource {
  target          = aws_cloudfront_function.spa_router
  override_during = plan
  values = {
    arn = "arn:aws:cloudfront::123456789012:function/plantrain-dev-spa-router"
  }
}

override_resource {
  target          = aws_cloudfront_origin_access_control.spa
  override_during = plan
  values = {
    id = "E2OACABCDEFGHI"
  }
}

override_resource {
  target          = aws_cloudfront_distribution.this
  override_during = plan
  values = {
    id             = "E1ABCDEFGHIJKL"
    arn            = "arn:aws:cloudfront::111122223333:distribution/E1ABCDEFGHIJKL"
    domain_name    = "d111111abcdef8.cloudfront.net"
    hosted_zone_id = "Z2FDTNDATAQYW2"
  }
}

# The real provider reports the validation record's domain at plan; the mock cannot.
override_resource {
  target          = aws_acm_certificate.cert
  override_during = plan
  values = {
    arn = "arn:aws:acm:us-east-1:111122223333:certificate/11111111-2222-3333-4444-555555555555"
    domain_validation_options = [{
      domain_name           = "app.example.com"
      resource_record_name  = "_x.app.example.com."
      resource_record_type  = "CNAME"
      resource_record_value = "_y.acm-validations.aws."
    }]
  }
}

variables {
  name_prefix     = "plantrain-dev"
  api_enabled     = true
  api_origin_host = "abc123.execute-api.eu-west-2.amazonaws.com"
  dns_mode        = "none"
}

run "the_bucket_is_private_and_only_cloudfront_reads_it" {
  command = plan

  assert {
    condition = (
      aws_s3_bucket_public_access_block.spa.block_public_acls &&
      aws_s3_bucket_public_access_block.spa.block_public_policy &&
      aws_s3_bucket_public_access_block.spa.ignore_public_acls &&
      aws_s3_bucket_public_access_block.spa.restrict_public_buckets
    )
    error_message = "every public-access block is on"
  }
  assert {
    condition     = aws_cloudfront_origin_access_control.spa.signing_behavior == "always" && aws_cloudfront_origin_access_control.spa.origin_access_control_origin_type == "s3"
    error_message = "the bucket is reached through an origin access control"
  }
  assert {
    condition     = one(one(aws_s3_bucket_server_side_encryption_configuration.spa.rule).apply_server_side_encryption_by_default).sse_algorithm == "AES256"
    error_message = "objects are encrypted at rest"
  }
  assert {
    condition     = one(data.aws_iam_policy_document.spa.statement[0].condition).variable == "AWS:SourceArn"
    error_message = "the bucket policy admits this distribution only"
  }
  assert {
    condition     = one(aws_cloudfront_distribution.this.default_cache_behavior).viewer_protocol_policy == "redirect-to-https"
    error_message = "plain HTTP is redirected"
  }
}

run "the_api_is_routed_under_api_uncached" {
  command = plan

  assert {
    condition     = length(aws_cloudfront_distribution.this.ordered_cache_behavior) == 1 && one(aws_cloudfront_distribution.this.ordered_cache_behavior).path_pattern == "/api/*"
    error_message = "/api/* has its own behaviour"
  }
  assert {
    condition     = one(aws_cloudfront_distribution.this.ordered_cache_behavior).cache_policy_id == "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
    error_message = "the API is never cached (Managed-CachingDisabled)"
  }
  assert {
    condition     = length(aws_cloudfront_distribution.this.origin) == 2
    error_message = "a bucket origin and an API origin"
  }
}

run "a_site_with_no_api_has_no_api_behaviour" {
  command = plan
  variables {
    api_enabled     = false
    api_origin_host = ""
  }

  assert {
    condition     = length(aws_cloudfront_distribution.this.ordered_cache_behavior) == 0 && length(aws_cloudfront_distribution.this.origin) == 1
    error_message = "no http block means no /api/* route and no second origin"
  }
}

run "a_single_page_app_routes_through_an_edge_function_not_error_rewrites" {
  command = plan

  assert {
    condition     = length(aws_cloudfront_distribution.this.custom_error_response) == 0
    error_message = "custom_error_response is distribution-wide: it would answer the shell for /api/<unknown> and a missing asset"
  }
  assert {
    condition     = length(aws_cloudfront_function.spa_router) == 1 && aws_cloudfront_function.spa_router[0].runtime == "cloudfront-js-2.0"
    error_message = "the SPA router is a viewer-request function"
  }
  assert {
    condition     = one(aws_cloudfront_distribution.this.default_cache_behavior[0].function_association).event_type == "viewer-request"
    error_message = "associated on the default (site) behaviour at viewer-request"
  }
  assert {
    condition     = strcontains(aws_cloudfront_function.spa_router[0].code, "request.uri = '/index.html';") && strcontains(aws_cloudfront_function.spa_router[0].code, "indexOf('/api/')") && strcontains(aws_cloudfront_function.spa_router[0].code, "indexOf('.')")
    error_message = "the router rewrites to the index, and leaves /api/ and file-like paths alone"
  }
}

run "spa_false_serves_real_404s_and_a_named_index" {
  command = plan
  variables {
    spa            = false
    index_document = "home.html"
  }

  assert {
    condition     = length(aws_cloudfront_distribution.this.custom_error_response) == 0 && length(aws_cloudfront_function.spa_router) == 0
    error_message = "no rewrite at all when the site is not a single-page app"
  }
  assert {
    condition     = length(aws_cloudfront_distribution.this.default_cache_behavior[0].function_association) == 0
    error_message = "and no function association"
  }
  assert {
    condition     = aws_cloudfront_distribution.this.default_root_object == "home.html"
    error_message = "the index document is the manifest's"
  }
}

run "the_router_rewrites_to_a_named_index" {
  command = plan
  variables {
    index_document = "app/home.html"
  }

  assert {
    condition     = strcontains(aws_cloudfront_function.spa_router[0].code, "request.uri = '/app/home.html';")
    error_message = "application routes serve the manifest's index document"
  }
}

run "the_security_headers_policy_is_ours_and_closed" {
  command = plan

  assert {
    condition     = strcontains(aws_cloudfront_response_headers_policy.security.security_headers_config[0].content_security_policy[0].content_security_policy, "default-src 'self'") && strcontains(aws_cloudfront_response_headers_policy.security.security_headers_config[0].content_security_policy[0].content_security_policy, "script-src 'self'") && strcontains(aws_cloudfront_response_headers_policy.security.security_headers_config[0].content_security_policy[0].content_security_policy, "frame-ancestors 'none'")
    error_message = "a CSP with default-src and script-src on 'self'"
  }
  assert {
    condition     = alltrue([for d in split("; ", aws_cloudfront_response_headers_policy.security.security_headers_config[0].content_security_policy[0].content_security_policy) : !strcontains(d, "'unsafe-inline'") || startswith(d, "style-src-attr ")])
    error_message = "no script 'unsafe-inline'"
  }
  assert {
    condition     = strcontains(aws_cloudfront_response_headers_policy.security.security_headers_config[0].content_security_policy[0].content_security_policy, "style-src-attr 'unsafe-inline'")
    error_message = "inline style attributes only"
  }
  assert {
    condition     = aws_cloudfront_response_headers_policy.security.security_headers_config[0].frame_options[0].frame_option == "DENY" && aws_cloudfront_response_headers_policy.security.security_headers_config[0].referrer_policy[0].referrer_policy == "strict-origin-when-cross-origin" && length(aws_cloudfront_response_headers_policy.security.security_headers_config[0].content_type_options) == 1
    error_message = "frame DENY, a strict referrer policy and nosniff"
  }
}

run "the_policy_is_on_both_behaviours" {
  command = plan
  variables {
    api_enabled     = true
    api_origin_host = "abc123.execute-api.eu-west-2.amazonaws.com"
  }

  assert {
    condition     = aws_cloudfront_distribution.this.default_cache_behavior[0].response_headers_policy_id == aws_cloudfront_response_headers_policy.security.id
    error_message = "the site carries the policy"
  }
  assert {
    condition     = one(aws_cloudfront_distribution.this.ordered_cache_behavior).response_headers_policy_id == aws_cloudfront_response_headers_policy.security.id
    error_message = "so does /api/*: an unheadered API is the half-fix"
  }
  assert {
    condition     = length(one(aws_cloudfront_distribution.this.ordered_cache_behavior).function_association) == 0
    error_message = "the router is never attached to /api/*"
  }
}

run "script_hashes_join_script_src_and_an_override_replaces_the_policy" {
  command = plan
  variables {
    csp_script_hashes = "'sha256-abc123=' 'sha256-def456='"
  }

  assert {
    condition     = strcontains(aws_cloudfront_response_headers_policy.security.security_headers_config[0].content_security_policy[0].content_security_policy, "script-src 'self' 'sha256-abc123=' 'sha256-def456='; ")
    error_message = "the inline bootstrap is admitted by hash"
  }
}

run "an_override_replaces_the_whole_policy" {
  command = plan
  variables {
    content_security_policy = "default-src 'none'"
  }

  assert {
    condition     = aws_cloudfront_response_headers_policy.security.security_headers_config[0].content_security_policy[0].content_security_policy == "default-src 'none'"
    error_message = "an explicit policy wins outright"
  }
}

run "a_script_hash_must_be_a_quoted_hash_source" {
  command = plan
  variables {
    csp_script_hashes = "'unsafe-inline'"
  }
  expect_failures = [var.csp_script_hashes]
}

run "dns_none_creates_no_dns_or_certificate" {
  command = plan

  assert {
    condition     = length(aws_route53_zone.app) == 0 && length(aws_acm_certificate.cert) == 0 && length(aws_route53_record.alias_a) == 0
    error_message = "none states itself: the CloudFront URL, no Route 53, no ACM"
  }
  assert {
    condition     = one(aws_cloudfront_distribution.this.viewer_certificate).cloudfront_default_certificate == true
    error_message = "the default CloudFront certificate"
  }
  assert {
    condition     = output.app_url == "https://${aws_cloudfront_distribution.this.domain_name}"
    error_message = "the app is served on the distribution URL"
  }
}

run "external_delegated_first_apply_never_waits_for_validation" {
  command = plan
  variables {
    dns_mode    = "external_delegated"
    domain_name = "app.example.com"
  }

  assert {
    condition     = length(aws_route53_zone.app) == 1 && length(aws_acm_certificate.cert) == 1
    error_message = "the child zone and the certificate exist from the first apply"
  }
  assert {
    condition     = length(aws_acm_certificate_validation.cert) == 0 && length(aws_route53_record.alias_a) == 0 && length(aws_route53_record.delegation) == 0
    error_message = "no validation wait, no alias and no delegation record before domain_ready: the first apply cannot hang on a delegation that is not live (ADR-0041)"
  }
  assert {
    condition     = one(aws_cloudfront_distribution.this.viewer_certificate).cloudfront_default_certificate == true && length(aws_cloudfront_distribution.this.aliases) == 0
    error_message = "the distribution keeps its default certificate until the domain is attached"
  }
}

run "external_delegated_attaches_once_ready" {
  command = plan
  variables {
    dns_mode     = "external_delegated"
    domain_name  = "app.example.com"
    domain_ready = true
  }

  assert {
    condition     = length(aws_acm_certificate_validation.cert) == 1 && length(aws_route53_record.alias_a) == 1 && length(aws_route53_record.alias_aaaa) == 1
    error_message = "the cert is awaited and the alias records exist"
  }
  assert {
    condition     = length(aws_route53_record.delegation) == 0
    error_message = "the operator delegates by hand in this mode"
  }
  assert {
    condition     = output.app_url == "https://app.example.com"
    error_message = "the app is served on its domain"
  }
}

run "in_account_writes_the_delegation_itself" {
  command = plan
  variables {
    dns_mode       = "in_account"
    domain_name    = "app.example.com"
    parent_zone_id = "Z0123456789ABCDEFGHIJ"
    domain_ready   = true
  }

  assert {
    condition     = length(aws_route53_record.delegation) == 1 && aws_route53_record.delegation[0].zone_id == "Z0123456789ABCDEFGHIJ" && aws_route53_record.delegation[0].type == "NS"
    error_message = "the NS delegation is written into the parent zone"
  }
}

run "in_account_delegates_before_it_is_ready" {
  command = plan
  variables {
    dns_mode       = "in_account"
    domain_name    = "app.example.com"
    parent_zone_id = "Z0123456789ABCDEFGHIJ"
  }

  assert {
    condition     = length(aws_route53_record.delegation) == 1 && length(aws_acm_certificate_validation.cert) == 0
    error_message = "the delegation is written on the first apply (validation needs it) while the attach waits for domain_ready"
  }
}

run "the_certificate_lives_in_us_east_1" {
  command = plan
  variables {
    dns_mode    = "external_delegated"
    domain_name = "app.example.com"
  }

  assert {
    condition     = aws_acm_certificate.cert[0].validation_method == "DNS" && aws_acm_certificate.cert[0].domain_name == "app.example.com"
    error_message = "a DNS-validated certificate for the domain"
  }
}

run "dns_mode_has_no_default_and_no_other_value" {
  command = plan
  variables {
    dns_mode = "route66"
  }
  expect_failures = [var.dns_mode]
}

run "a_custom_domain_needs_a_domain_name" {
  command = plan
  variables {
    dns_mode = "external_delegated"
  }
  expect_failures = [var.domain_name]
}

run "a_domain_name_must_be_a_plain_hostname" {
  command = plan
  variables {
    dns_mode    = "external_delegated"
    domain_name = "app.example.com\" ; evil"
  }
  expect_failures = [var.domain_name]
}

run "in_account_needs_the_parent_zone" {
  command = plan
  variables {
    dns_mode    = "in_account"
    domain_name = "app.example.com"
  }
  expect_failures = [var.parent_zone_id]
}

run "an_index_document_cannot_climb_out" {
  command = plan
  variables {
    index_document = "../secret.html"
  }
  expect_failures = [var.index_document]
}
