# Offline: the AWS provider is mocked. These assert what the module plans for services.auth.
mock_provider "aws" {}

variables {
  name_prefix   = "plantrain-dev"
  provider_name = "cognito"
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
