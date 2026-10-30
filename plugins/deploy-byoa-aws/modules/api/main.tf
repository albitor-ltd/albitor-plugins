data "aws_caller_identity" "current" {}

locals {
  # Where the app's secrets are read from, handed to the function so the code and the grant below
  # name the same path. Empty prefix: no env var and no grant.
  secrets_env = var.secrets_parameter_prefix == "" ? {} : {
    ALBITOR_SECRETS_PARAMETER_PREFIX = var.secrets_parameter_prefix
  }

  # The official AWS Lambda Web Adapter layer (publisher account 753240598075), x86_64, at a pinned
  # version. The API is a plain net/http server; the adapter forwards API Gateway events to it.
  adapter_layer_arn = coalesce(
    var.lambda_adapter_layer_arn,
    "arn:aws:lambda:${var.region}:753240598075:layer:LambdaAdapterLayerX86:24",
  )

  # The adapter's own settings come last, so nothing the manifest or the data module passes can
  # override them (the variable also refuses those names).
  environment = merge(var.environment, local.secrets_env, {
    PORT                         = "8080"
    AWS_LWA_PORT                 = "8080"
    AWS_LWA_READINESS_CHECK_PATH = "/api/health"
    AWS_LAMBDA_EXEC_WRAPPER      = "/opt/bootstrap"
  })
}

# The pipeline builds the binary before it plans; terraform validate and test never read it.
# The zip carries the file as 0755: an artifact download drops the mode bit, and a Lambda that
# cannot execute its handler fails at invoke time, not at apply.
data "archive_file" "api" {
  type             = "zip"
  source_file      = "${var.build_root}/${var.binary}"
  output_path      = "${path.root}/.terraform/build/api.zip"
  output_file_mode = "0755"
}

resource "aws_iam_role" "lambda" {
  name = "${var.name_prefix}-api"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "logs" {
  role       = aws_iam_role.lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# The data-dynamodb store reads and writes items and queries one partition; it never scans, so the
# role cannot. Index ARNs are named so an index can be queried, nothing else is granted.
data "aws_iam_policy_document" "tables" {
  count = length(var.table_arns) == 0 ? 0 : 1

  statement {
    sid = "Tables"
    actions = [
      "dynamodb:BatchGetItem",
      "dynamodb:BatchWriteItem",
      "dynamodb:ConditionCheckItem",
      "dynamodb:DeleteItem",
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:Query",
      "dynamodb:UpdateItem",
    ]
    resources = concat(var.table_arns, [for arn in var.table_arns : "${arn}/index/*"])
  }
}

resource "aws_iam_role_policy" "tables" {
  count = length(var.table_arns) == 0 ? 0 : 1

  name   = "tables"
  role   = aws_iam_role.lambda.id
  policy = data.aws_iam_policy_document.tables[0].json
}

# The platform writes each app secret as an SSM SecureString at <prefix><NAME> (ADR-0057). The
# function may read exactly those parameters and nothing wider: no GetParametersByPath, no write.
# Parameters sealed with the AWS-managed alias/aws/ssm key need no grant of their own (its key policy
# admits the account's principals through SSM); a customer-managed key needs kms:Decrypt on it.
data "aws_iam_policy_document" "secrets" {
  count = var.secrets_parameter_prefix == "" ? 0 : 1

  statement {
    sid       = "SecretsRead"
    actions   = ["ssm:GetParameter", "ssm:GetParameters"]
    resources = ["arn:aws:ssm:${var.region}:${data.aws_caller_identity.current.account_id}:parameter${var.secrets_parameter_prefix}*"]
  }

  dynamic "statement" {
    for_each = var.secrets_kms_key_arn == null ? [] : [var.secrets_kms_key_arn]
    content {
      sid       = "SecretsDecrypt"
      actions   = ["kms:Decrypt"]
      resources = [statement.value]
    }
  }
}

resource "aws_iam_role_policy" "secrets" {
  count = var.secrets_parameter_prefix == "" ? 0 : 1

  name   = "secrets"
  role   = aws_iam_role.lambda.id
  policy = data.aws_iam_policy_document.secrets[0].json
}

resource "aws_cloudwatch_log_group" "api" {
  name              = "/aws/lambda/${var.name_prefix}-api"
  retention_in_days = var.log_retention_days
  tags              = var.tags
}

resource "aws_lambda_function" "api" {
  function_name    = "${var.name_prefix}-api"
  role             = aws_iam_role.lambda.arn
  filename         = data.archive_file.api.output_path
  source_code_hash = data.archive_file.api.output_base64sha256
  runtime          = "provided.al2023"
  handler          = basename(var.binary)
  architectures    = ["x86_64"]
  memory_size      = var.lambda_memory_mb
  timeout          = var.lambda_timeout_s
  layers           = [local.adapter_layer_arn]

  environment {
    variables = local.environment
  }

  depends_on = [aws_cloudwatch_log_group.api, aws_iam_role_policy_attachment.logs]
  tags       = var.tags
}

# One HTTP API with the single $default route: CloudFront sends /api/* here and the app's own
# router owns every path below it.
resource "aws_apigatewayv2_api" "api" {
  name          = "${var.name_prefix}-api"
  protocol_type = "HTTP"
  tags          = var.tags
}

resource "aws_apigatewayv2_integration" "api" {
  api_id                 = aws_apigatewayv2_api.api.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.api.invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "default" {
  api_id    = aws_apigatewayv2_api.api.id
  route_key = "$default"
  target    = "integrations/${aws_apigatewayv2_integration.api.id}"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.api.id
  name        = "$default"
  auto_deploy = true

  default_route_settings {
    throttling_rate_limit  = var.throttle_rate
    throttling_burst_limit = var.throttle_burst
  }

  tags = var.tags
}

resource "aws_lambda_permission" "apigw" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.api.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.api.execution_arn}/*/*"
}
