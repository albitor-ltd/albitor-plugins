mock_provider "aws" {}

variables {
  name_prefix   = "plantrain-dev"
  api_enabled   = true
  function_name = "plantrain-dev-api"
  api_id        = "abc123"
  table_names   = { data = "plantrain-dev-data" }
}

run "an_api_gets_error_throttle_and_gateway_alarms" {
  command = plan

  assert {
    condition     = aws_cloudwatch_metric_alarm.lambda_errors[0].metric_name == "Errors" && aws_cloudwatch_metric_alarm.lambda_errors[0].dimensions["FunctionName"] == "plantrain-dev-api"
    error_message = "the function's errors are alarmed on, by name"
  }
  assert {
    condition     = aws_cloudwatch_metric_alarm.lambda_throttles[0].metric_name == "Throttles"
    error_message = "the function's throttles are alarmed on"
  }
  assert {
    condition     = aws_cloudwatch_metric_alarm.gateway_5xx[0].metric_name == "5xx" && aws_cloudwatch_metric_alarm.gateway_5xx[0].dimensions["ApiId"] == "abc123"
    error_message = "the gateway's server errors are alarmed on"
  }
}

run "every_table_is_watched_for_throttling" {
  command = plan
  variables {
    table_names = { data = "plantrain-dev-data", audit-log = "plantrain-dev-audit-log" }
  }

  assert {
    condition     = length(aws_cloudwatch_metric_alarm.table_read_throttles) == 2 && length(aws_cloudwatch_metric_alarm.table_write_throttles) == 2
    error_message = "read and write throttle alarms for each table"
  }
  assert {
    condition     = aws_cloudwatch_metric_alarm.table_write_throttles["audit-log"].dimensions["TableName"] == "plantrain-dev-audit-log"
    error_message = "by the AWS table name"
  }
}

run "an_alarm_notifies_the_topic_and_missing_data_is_healthy" {
  command = plan

  assert {
    condition     = alltrue([for a in [aws_cloudwatch_metric_alarm.lambda_errors[0], aws_cloudwatch_metric_alarm.gateway_5xx[0]] : a.treat_missing_data == "notBreaching" && length(a.alarm_actions) == 1])
    error_message = "no traffic is not an incident, and an alarm notifies the topic"
  }
}

run "no_address_means_a_topic_with_no_subscriber" {
  command = plan

  assert {
    condition     = length(aws_sns_topic_subscription.email) == 0
    error_message = "no email, no subscription: the topic is there to attach one to"
  }
}

run "an_address_is_subscribed" {
  command = plan
  variables {
    alarm_email = "ops@example.com"
  }

  assert {
    condition     = aws_sns_topic_subscription.email[0].protocol == "email" && aws_sns_topic_subscription.email[0].endpoint == "ops@example.com"
    error_message = "the address is subscribed to the topic"
  }
}

run "no_api_means_no_api_alarms" {
  command = plan
  variables {
    api_enabled   = false
    function_name = ""
    api_id        = ""
    table_names   = {}
  }

  assert {
    condition     = length(aws_cloudwatch_metric_alarm.lambda_errors) == 0 && length(aws_cloudwatch_metric_alarm.gateway_5xx) == 0 && length(aws_cloudwatch_metric_alarm.table_read_throttles) == 0
    error_message = "a static site has nothing to alarm on"
  }
}

run "a_malformed_address_is_refused" {
  command = plan
  variables {
    alarm_email = "x@y.com,evil@z.com"
  }
  expect_failures = [var.alarm_email]
}
