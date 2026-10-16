# The user pool behind services.auth, shaped for the auth-cognito plugin's API: it signs people in
# with the email address as the username (USER_PASSWORD_AUTH from a public client, so the browser
# never calls Cognito), verifies the ID and access tokens against the pool's JWKS, and reads
# cognito:groups. The posture (private or public) is enforced by the app's own routes and changes
# nothing here, so it is not an input.
resource "aws_cognito_user_pool" "this" {
  # Derived from the app's prefix; the manifest never names an AWS resource.
  name = "${var.name_prefix}-users"

  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  # The auth-cognito API lowercases the email it signs in with, so the pool must not tell
  # Ada@x.com from ada@x.com. Immutable once the pool exists.
  username_configuration {
    case_sensitive = false
  }

  # invite_only: the public SignUp call is refused, so accounts are created by an administrator
  # (the deploy seeds the platform accounts, and the invite workflow adds people; see the README).
  admin_create_user_config {
    allow_admin_create_user_only = var.sign_up == "invite_only"
  }

  password_policy {
    minimum_length                   = 12
    require_lowercase                = true
    require_uppercase                = true
    require_numbers                  = true
    require_symbols                  = false
    temporary_password_validity_days = 7
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  mfa_configuration = "OFF"

  # Removing the pool deletes every account in it, so a manifest edit must not do it quietly.
  deletion_protection = var.deletion_protection ? "ACTIVE" : "INACTIVE"

  tags = var.tags
}

# A public client with no secret: nothing server-side could keep one from a browser-facing flow.
# PreventUserExistenceErrors is required by the auth-cognito plugin, which cannot set it: without
# it Cognito tells an attacker which email addresses have accounts.
resource "aws_cognito_user_pool_client" "this" {
  name         = "${var.name_prefix}-app"
  user_pool_id = aws_cognito_user_pool.this.id

  generate_secret               = false
  prevent_user_existence_errors = "ENABLED"
  enable_token_revocation       = true

  explicit_auth_flows = ["ALLOW_USER_PASSWORD_AUTH", "ALLOW_REFRESH_TOKEN_AUTH"]

  access_token_validity  = 1
  id_token_validity      = 1
  refresh_token_validity = 30
  token_validity_units {
    access_token  = "hours"
    id_token      = "hours"
    refresh_token = "days"
  }
}

resource "aws_cognito_user_group" "this" {
  for_each = toset(var.groups)

  name         = each.value
  user_pool_id = aws_cognito_user_pool.this.id
}
