mock_provider "aws" {
  mock_data "aws_iam_policy_document" {
    defaults = {
      json = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
    }
  }
}

mock_provider "archive" {}

variables {
  name_prefix = "plantrain-dev"
  region      = "eu-west-2"
  build_root  = "../.."
  binary      = "api/dist/api"
  environment = { ALBITOR_TABLE_DATA = "plantrain-dev-data" }
  table_arns  = ["arn:aws:dynamodb:eu-west-2:111122223333:table/plantrain-dev-data"]
}

run "the_function_runs_the_built_binary_behind_the_web_adapter" {
  command = plan

  assert {
    condition     = aws_lambda_function.api.runtime == "provided.al2023" && aws_lambda_function.api.handler == "api"
    error_message = "a custom runtime started through the adapter: the handler is the binary's file name"
  }
  assert {
    condition     = one(aws_lambda_function.api.layers) == "arn:aws:lambda:eu-west-2:753240598075:layer:LambdaAdapterLayerX86:24"
    error_message = "the Lambda Web Adapter layer for the region, at a pinned version"
  }
  assert {
    condition     = one(aws_lambda_function.api.architectures) == "x86_64"
    error_message = "x86_64, matching the layer and the workflow's GOARCH"
  }
  assert {
    condition = (
      aws_lambda_function.api.environment[0].variables["AWS_LAMBDA_EXEC_WRAPPER"] == "/opt/bootstrap" &&
      aws_lambda_function.api.environment[0].variables["AWS_LWA_PORT"] == "8080" &&
      aws_lambda_function.api.environment[0].variables["AWS_LWA_READINESS_CHECK_PATH"] == "/api/health"
    )
    error_message = "the adapter is started, pointed at the server's port and its health route"
  }
  assert {
    condition     = aws_lambda_function.api.environment[0].variables["ALBITOR_TABLE_DATA"] == "plantrain-dev-data"
    error_message = "the table environment reaches the function"
  }
  assert {
    condition     = data.archive_file.api.output_file_mode == "0755"
    error_message = "the binary is executable in the zip: an artifact download drops the mode bit"
  }
}

run "the_function_has_only_the_table_permissions_it_needs" {
  command = plan

  assert {
    condition     = length(aws_iam_role_policy.tables) == 1
    error_message = "one inline policy for the tables"
  }
  assert {
    condition     = !contains(flatten(data.aws_iam_policy_document.tables[0].statement[*].actions), "*") && !contains(flatten(data.aws_iam_policy_document.tables[0].statement[*].actions), "dynamodb:*")
    error_message = "no wildcard action"
  }
  assert {
    condition     = !contains(flatten(data.aws_iam_policy_document.tables[0].statement[*].actions), "dynamodb:Scan")
    error_message = "the store queries one partition; it never scans"
  }
  assert {
    condition = sort(flatten(data.aws_iam_policy_document.tables[0].statement[*].resources)) == sort([
      "arn:aws:dynamodb:eu-west-2:111122223333:table/plantrain-dev-data",
      "arn:aws:dynamodb:eu-west-2:111122223333:table/plantrain-dev-data/index/*",
    ])
    error_message = "the statement names the tables and their indexes, not *"
  }
}

run "no_tables_means_no_table_policy" {
  command = plan
  variables {
    table_arns  = []
    environment = {}
  }

  assert {
    condition     = length(aws_iam_role_policy.tables) == 0
    error_message = "an app with no tables gets no DynamoDB permission"
  }
}

run "the_gateway_has_one_default_route_and_throttles_the_stage" {
  command = plan

  assert {
    condition     = aws_apigatewayv2_route.default.route_key == "$default" && aws_apigatewayv2_api.api.protocol_type == "HTTP"
    error_message = "an HTTP API with the single default route: the app's router owns every path below /api"
  }
  assert {
    condition     = aws_apigatewayv2_stage.default.default_route_settings[0].throttling_rate_limit == 50 && aws_apigatewayv2_stage.default.default_route_settings[0].throttling_burst_limit == 100
    error_message = "the stage is throttled by default"
  }
  assert {
    condition     = aws_cloudwatch_log_group.api.retention_in_days == 14
    error_message = "logs are kept for fourteen days by default"
  }
}

run "the_adapter_layer_can_be_named" {
  command = plan
  variables {
    lambda_adapter_layer_arn = "arn:aws:lambda:eu-west-2:753240598075:layer:LambdaAdapterLayerX86:25"
  }

  assert {
    condition     = one(aws_lambda_function.api.layers) == "arn:aws:lambda:eu-west-2:753240598075:layer:LambdaAdapterLayerX86:25"
    error_message = "an explicit layer wins over the derived one"
  }
}

run "an_environment_variable_cannot_override_the_adapter_s" {
  command = plan
  variables {
    environment = { AWS_LAMBDA_EXEC_WRAPPER = "/tmp/evil" }
  }
  expect_failures = [var.environment]
}

run "an_environment_variable_Lambda_reserves_is_refused" {
  command = plan
  variables {
    environment = { AWS_REGION = "us-east-1" }
  }
  expect_failures = [var.environment]
}

run "a_binary_path_that_leaves_the_repository_is_refused" {
  command = plan
  variables {
    binary = "../../../etc/passwd"
  }
  expect_failures = [var.binary]
}

run "a_binary_path_with_a_shell_or_glob_character_is_refused" {
  command = plan
  variables {
    binary = "api/dist/api;x"
  }
  expect_failures = [var.binary]
}

run "memory_and_timeout_stay_inside_Lambda_s_limits" {
  command = plan
  variables {
    lambda_memory_mb = 10
  }
  expect_failures = [var.lambda_memory_mb]
}

run "an_adapter_layer_that_is_not_a_Lambda_layer_ARN_is_refused" {
  command = plan
  variables {
    lambda_adapter_layer_arn = "arn:aws:s3:::bucket"
  }
  expect_failures = [var.lambda_adapter_layer_arn]
}
