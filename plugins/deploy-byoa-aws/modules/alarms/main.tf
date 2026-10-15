locals {
  has_api = var.api_enabled
}

# The topic every alarm notifies. It exists with no subscriber when no address is given, so a team
# can attach its own (a chat webhook, a pager) without a module change.
resource "aws_sns_topic" "alarms" {
  name = "${var.name_prefix}-alarms"
  tags = var.tags
}

resource "aws_sns_topic_subscription" "email" {
  count = var.alarm_email == "" ? 0 : 1

  topic_arn = aws_sns_topic.alarms.arn
  protocol  = "email"
  endpoint  = var.alarm_email
}

# --- the API function and its gateway -------------------------------------------

resource "aws_cloudwatch_metric_alarm" "lambda_errors" {
  count = local.has_api ? 1 : 0

  alarm_name          = "${var.name_prefix}-api-errors"
  alarm_description   = "The API function returned errors."
  namespace           = "AWS/Lambda"
  metric_name         = "Errors"
  dimensions          = { FunctionName = var.function_name }
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = [aws_sns_topic.alarms.arn]
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "lambda_throttles" {
  count = local.has_api ? 1 : 0

  alarm_name          = "${var.name_prefix}-api-throttles"
  alarm_description   = "The API function was throttled: concurrency is exhausted."
  namespace           = "AWS/Lambda"
  metric_name         = "Throttles"
  dimensions          = { FunctionName = var.function_name }
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = [aws_sns_topic.alarms.arn]
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "gateway_5xx" {
  count = local.has_api ? 1 : 0

  alarm_name          = "${var.name_prefix}-gateway-5xx"
  alarm_description   = "The API gateway returned server errors."
  namespace           = "AWS/ApiGateway"
  metric_name         = "5xx"
  dimensions          = { ApiId = var.api_id }
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = [aws_sns_topic.alarms.arn]
  tags                = var.tags
}

# --- the tables ---------------------------------------------------------------

resource "aws_cloudwatch_metric_alarm" "table_read_throttles" {
  for_each = var.table_names

  alarm_name          = "${var.name_prefix}-${each.key}-read-throttles"
  alarm_description   = "Reads on table ${each.key} were throttled."
  namespace           = "AWS/DynamoDB"
  metric_name         = "ReadThrottleEvents"
  dimensions          = { TableName = each.value }
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = [aws_sns_topic.alarms.arn]
  tags                = var.tags
}

resource "aws_cloudwatch_metric_alarm" "table_write_throttles" {
  for_each = var.table_names

  alarm_name          = "${var.name_prefix}-${each.key}-write-throttles"
  alarm_description   = "Writes on table ${each.key} were throttled."
  namespace           = "AWS/DynamoDB"
  metric_name         = "WriteThrottleEvents"
  dimensions          = { TableName = each.value }
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = [aws_sns_topic.alarms.arn]
  tags                = var.tags
}
