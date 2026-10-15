output "topic_arn" {
  description = "The SNS topic every alarm notifies. Subscribe a chat webhook or a pager to it."
  value       = aws_sns_topic.alarms.arn
}
