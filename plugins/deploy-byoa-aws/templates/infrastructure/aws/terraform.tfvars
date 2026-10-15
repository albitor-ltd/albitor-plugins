# Per-environment choices for the deployed stack. Terraform loads this file by name. It is yours:
# the plugin seeded it once and never rewrites it. What the stack contains comes from albitor.yaml;
# only how it is served and who hears about problems is decided here.
#
# project_name and environment are set by the deploy workflow (from the repository name and the
# `dev` GitHub environment); set either here to override.

# How the app is served. There is no default: pick one (ADR-0041).
#   none               the CloudFront URL, no Route 53 or ACM. Start here: zero DNS wiring.
#   in_account         the parent zone is in this AWS account; Terraform wires everything, and
#                      domain_ready may be true from the start.
#   external_delegated the parent zone is elsewhere. The first apply creates the child zone and the
#                      certificate and prints route53_nameservers; delegate them from the parent,
#                      then set domain_ready = true and apply again.
dns_mode = "none"

# domain_name    = "app.example.com"
# parent_zone_id = "Z0123456789ABCDEFGHIJ" # in_account only: the zone that holds the delegation
# domain_ready   = false

# An address to tell when an alarm fires (the API erroring or being throttled, a table throttled).
# AWS emails a confirmation link that must be accepted. Empty subscribes nobody; the topic is the
# alarm_topic_arn output, so a chat webhook or a pager can subscribe to it instead.
alarm_email = ""
