# The user pool behind services.auth, shaped for the auth-cognito plugin's API: it signs people in
# with the email address as the username (USER_PASSWORD_AUTH from a public client, so the browser
# never calls Cognito), verifies the ID and access tokens against the pool's JWKS, and reads
# cognito:groups. The posture (private or public) is enforced by the app's own routes and changes
# nothing here, so it is not an input.
resource "aws_cognito_user_pool" "this" {
  # Derived from the app's prefix; the manifest never names an AWS resource.
  name = "${var.name_prefix}-users"

  username_attributes = ["email"]

  # open: the pool emails a confirmation code, so email is auto-verified. invite_only sends no
  # email at all: people are created by an administrator with a suppressed invitation and a
  # temporary password handed over out of band, so there is nothing to verify by email and no
  # forgot-password mail (see account_recovery_setting). Both are in-place pool updates.
  auto_verified_attributes = var.sign_up == "open" ? ["email"] : []

  # The auth-cognito API lowercases the email it signs in with, so the pool must not tell
  # Ada@x.com from ada@x.com. Immutable once the pool exists.
  username_configuration {
    case_sensitive = false
  }

  # invite_only: the public SignUp call is refused, so accounts are created by an administrator
  # (the deploy seeds the platform accounts, and the invite workflow adds people; see the README).
  admin_create_user_config {
    allow_admin_create_user_only = var.sign_up == "invite_only"

    # The invitation says what the app is and where to sign in: Cognito's default is a bare
    # "Your username is {username} and temporary password is {####}" from nobody in particular.
    invite_message_template {
      email_subject = "You have been invited to ${var.app_name}"
      email_message = "You have been invited to ${var.app_name}.${var.sign_in_url == "" ? "" : " Sign in at ${var.sign_in_url}."} Your username is {username} and your temporary password is {####}. You will be asked to choose a password of your own the first time you sign in. The temporary password expires in 7 days."
      sms_message   = "${var.app_name}: your username is {username} and your temporary password is {####}."
    }
  }

  password_policy {
    minimum_length                   = 12
    require_lowercase                = true
    require_uppercase                = true
    require_numbers                  = true
    require_symbols                  = false
    temporary_password_validity_days = 7
  }

  # invite_only: no self-service recovery. An administrator resets a password (the People screen,
  # or the recover-owner workflow for the owner), so Cognito never sends a recovery code.
  account_recovery_setting {
    recovery_mechanism {
      name     = var.sign_up == "open" ? "verified_email" : "admin_only"
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

# invite_only: the group that may use the People screen. It is separate from admin_group, which is
# an app role, so who administers people is never an accident of how the app names its roles.
resource "aws_cognito_user_group" "user_admin" {
  count = var.sign_up == "invite_only" ? 1 : 0

  name         = var.user_admin_group
  user_pool_id = aws_cognito_user_pool.this.id
}

resource "aws_cognito_user_group" "this" {
  for_each = toset(var.groups)

  name         = each.value
  user_pool_id = aws_cognito_user_pool.this.id
}
