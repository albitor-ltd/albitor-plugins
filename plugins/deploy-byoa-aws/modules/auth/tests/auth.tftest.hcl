# Offline: the AWS provider is mocked. These assert what the module plans for services.auth.
mock_provider "aws" {}

variables {
  name_prefix   = "plantrain-dev"
  provider_name = "cognito"
  app_name      = "plantrain"
}

run "invite_only_is_the_default_and_the_client_is_public_and_hardened" {
  command = plan

  assert {
    condition     = aws_cognito_user_pool.this.name == "plantrain-dev-users"
    error_message = "the AWS name is derived from the prefix"
  }
  assert {
    condition     = one(aws_cognito_user_pool.this.admin_create_user_config).allow_admin_create_user_only
    error_message = "sign_up defaults to invite_only: only an administrator creates accounts"
  }
  assert {
    condition     = aws_cognito_user_pool.this.deletion_protection == "ACTIVE"
    error_message = "the pool is protected from deletion by default"
  }
  assert {
    condition     = contains(aws_cognito_user_pool.this.username_attributes, "email")
    error_message = "the email address is the username, as the auth-cognito API signs in"
  }
  assert {
    condition     = !one(aws_cognito_user_pool.this.username_configuration).case_sensitive
    error_message = "usernames are case-insensitive: the API lowercases emails and the setting cannot change later"
  }
  assert {
    condition     = aws_cognito_user_pool_client.this.prevent_user_existence_errors == "ENABLED"
    error_message = "PreventUserExistenceErrors is on, or sign-in leaks which accounts exist"
  }
  assert {
    condition     = !aws_cognito_user_pool_client.this.generate_secret
    error_message = "a public client"
  }
  assert {
    condition     = contains(aws_cognito_user_pool_client.this.explicit_auth_flows, "ALLOW_USER_PASSWORD_AUTH") && contains(aws_cognito_user_pool_client.this.explicit_auth_flows, "ALLOW_REFRESH_TOKEN_AUTH")
    error_message = "the API signs in with USER_PASSWORD_AUTH and refreshes with the refresh token"
  }
  assert {
    condition     = length(aws_cognito_user_group.this) == 0
    error_message = "no groups unless the manifest names them"
  }
}

run "open_sign_up_lets_anyone_register" {
  command = plan
  variables {
    sign_up = "open"
  }
  assert {
    condition     = !one(aws_cognito_user_pool.this.admin_create_user_config).allow_admin_create_user_only
    error_message = "open sign-up"
  }
  assert {
    condition     = aws_cognito_user_pool.this.auto_verified_attributes == toset(["email"])
    error_message = "open mode keeps email auto-verification: the confirmation code is emailed"
  }
  assert {
    condition     = one(one(aws_cognito_user_pool.this.account_recovery_setting).recovery_mechanism).name == "verified_email"
    error_message = "open mode keeps self-service recovery by verified email"
  }
  assert {
    condition     = output.env.AUTH_SIGN_UP == "open" && !output.invite_only
    error_message = "the API is told it is open"
  }
}

# These two changes are in-place updates on an existing pool: auto_verified_attributes and
# account_recovery_setting are not ForceNew in the AWS provider (only name-independent immutable
# schema such as username_attributes, username_configuration and schema force replacement, and
# none of those change here), and the open-mode values above are byte-for-byte what v0.7.0
# planned, so an `open` pool sees no diff at all.
run "invite_only_sends_no_auth_email_and_recovery_is_admin_only" {
  command = plan

  assert {
    condition     = length(aws_cognito_user_pool.this.auto_verified_attributes) == 0
    error_message = "no attribute is auto-verified: no verification email is ever sent"
  }
  assert {
    condition     = length(one(aws_cognito_user_pool.this.account_recovery_setting).recovery_mechanism) == 1 && one(one(aws_cognito_user_pool.this.account_recovery_setting).recovery_mechanism).name == "admin_only" && one(one(aws_cognito_user_pool.this.account_recovery_setting).recovery_mechanism).priority == 1
    error_message = "self-service recovery is off; an administrator resets passwords"
  }
  assert {
    condition     = contains(aws_cognito_user_pool.this.username_attributes, "email") && !one(aws_cognito_user_pool.this.username_configuration).case_sensitive
    error_message = "the immutable identity settings are unchanged, so the pool is updated in place, not replaced"
  }
}

run "invite_only_creates_the_people_admin_group_and_hands_it_to_the_api" {
  command = plan
  variables {
    groups = ["staff"]
  }
  assert {
    condition     = length(aws_cognito_user_group.user_admin) == 1 && aws_cognito_user_group.user_admin[0].name == "user-admins"
    error_message = "invite_only creates the dedicated people-admin group"
  }
  assert {
    condition     = output.user_admin_group == "user-admins" && output.env.AUTH_USER_ADMIN_GROUP == "user-admins"
    error_message = "the group is an output and in the function's environment"
  }
  assert {
    condition     = output.admin_group == "staff" && length(aws_cognito_user_group.this) == 1
    error_message = "the app's own groups and admin role are unchanged"
  }
}

run "open_mode_has_no_people_admin_group" {
  command = plan
  variables {
    sign_up = "open"
  }
  assert {
    condition     = length(aws_cognito_user_group.user_admin) == 0 && output.user_admin_group == "" && output.env.AUTH_USER_ADMIN_GROUP == ""
    error_message = "open: nothing administers people, so no group exists and the environment names none"
  }
}

run "the_people_admin_group_cannot_collide_with_an_app_group" {
  command = plan
  variables {
    groups           = ["staff", "people"]
    user_admin_group = "people"
  }
  expect_failures = [var.user_admin_group]
}

run "a_bad_people_admin_group_name_is_refused" {
  command = plan
  variables {
    user_admin_group = "Bad_Name"
  }
  expect_failures = [var.user_admin_group]
}

run "env_tells_the_api_the_mode_and_the_admin_group" {
  command = plan
  variables {
    groups = ["staff", "reviewer"]
  }
  assert {
    condition     = output.env.AUTH_SIGN_UP == "invite_only" && output.env.AUTH_ADMIN_GROUP == "staff" && output.invite_only
    error_message = "AUTH_SIGN_UP is the mode and AUTH_ADMIN_GROUP the resolved admin group"
  }
}

run "groups_become_pool_groups" {
  command = plan
  variables {
    groups = ["admin", "reviewer"]
  }
  assert {
    condition     = length(aws_cognito_user_group.this) == 2 && aws_cognito_user_group.this["admin"].name == "admin"
    error_message = "one group per name"
  }
}

run "admin_group_defaults_to_the_first_group" {
  command = plan
  variables {
    groups = ["staff", "reviewer"]
  }
  assert {
    condition     = output.admin_group == "staff"
    error_message = "an unset admin_group resolves to groups[0]"
  }
}

run "admin_group_is_empty_without_groups" {
  command = plan
  assert {
    condition     = output.admin_group == ""
    error_message = "an app with no groups has no admin group"
  }
}

run "a_declared_admin_group_wins" {
  command = plan
  variables {
    groups      = ["staff", "reviewer"]
    admin_group = "reviewer"
  }
  assert {
    condition     = output.admin_group == "reviewer"
    error_message = "a declared admin_group is the resolved one"
  }
}

run "an_admin_group_outside_groups_is_refused" {
  command = plan
  variables {
    groups      = ["staff"]
    admin_group = "owner"
  }
  expect_failures = [var.admin_group]
}

run "an_admin_group_with_no_groups_is_refused" {
  command = plan
  variables {
    admin_group = "admin"
  }
  expect_failures = [var.admin_group]
}

run "another_provider_is_refused" {
  command = plan
  variables {
    provider_name = "okta"
  }
  expect_failures = [var.provider_name]
}

run "an_unknown_sign_up_mode_is_refused" {
  command = plan
  variables {
    sign_up = "anyone"
  }
  expect_failures = [var.sign_up]
}

run "a_bad_group_name_is_refused" {
  command = plan
  variables {
    groups = ["Admin"]
  }
  expect_failures = [var.groups]
}

run "duplicate_groups_are_refused" {
  command = plan
  variables {
    groups = ["admin", "admin"]
  }
  expect_failures = [var.groups]
}

run "the_invitation_names_the_app_and_where_to_sign_in" {
  command = plan

  variables {
    sign_in_url = "https://plantrain.example.com"
  }

  assert {
    condition     = one(one(aws_cognito_user_pool.this.admin_create_user_config).invite_message_template).email_subject == "You have been invited to plantrain"
    error_message = "the invitation subject names the app"
  }
  assert {
    condition = alltrue([
      for token in ["{username}", "{####}", "plantrain", "https://plantrain.example.com"] :
      strcontains(one(one(aws_cognito_user_pool.this.admin_create_user_config).invite_message_template).email_message, token)
    ])
    error_message = "the invitation carries the username and temporary password placeholders, the app name and the sign-in URL"
  }
}

run "an_unknown_sign_in_url_leaves_the_link_out_but_keeps_the_placeholders" {
  command = plan

  assert {
    condition     = !strcontains(one(one(aws_cognito_user_pool.this.admin_create_user_config).invite_message_template).email_message, "Sign in at")
    error_message = "no link is invented when the URL is not known"
  }
  assert {
    condition     = strcontains(one(one(aws_cognito_user_pool.this.admin_create_user_config).invite_message_template).email_message, "{####}")
    error_message = "the temporary password placeholder is mandatory"
  }
}

run "a_hostile_sign_in_url_is_refused" {
  command = plan

  variables {
    sign_in_url = "https://x.example.com/ {####} javascript:"
  }

  expect_failures = [var.sign_in_url]
}

run "signup_domains_are_handed_to_the_function_and_default_to_none" {
  command = plan
  variables {
    sign_up        = "open"
    signup_domains = ["example.org", "*.example.co.uk"]
  }
  assert {
    condition     = output.env.AUTH_SIGNUP_DOMAINS == "example.org,*.example.co.uk"
    error_message = "the allow-list is comma-joined into AUTH_SIGNUP_DOMAINS"
  }
}

run "no_signup_domains_means_no_limit" {
  command = plan
  assert {
    condition     = output.env.AUTH_SIGNUP_DOMAINS == ""
    error_message = "unset is no limit"
  }
}

run "a_malformed_signup_domain_is_refused" {
  command = plan
  variables {
    signup_domains = ["not a domain"]
  }
  expect_failures = [var.signup_domains]
}
