###############################################################################
# SNS Topic
###############################################################################
resource "aws_sns_topic" "lambda_error" {
  name = "jishinranking-lambda-error"
}

resource "aws_sns_topic_subscription" "lambda_error_email" {
  topic_arn = aws_sns_topic.lambda_error.arn
  protocol  = "email"
  endpoint  = var.alert_email
}

###############################################################################
# CloudWatch Alarms
###############################################################################
resource "aws_cloudwatch_metric_alarm" "batch_error" {
  alarm_name          = "jishinranking-batch-error"
  alarm_description   = "Lambda jishinranking-batch でエラーが発生しました"
  namespace           = "AWS/Lambda"
  metric_name         = "Errors"
  dimensions = {
    FunctionName = aws_lambda_function.batch.function_name
  }

  statistic           = "Sum"
  period              = 60
  evaluation_periods  = 1
  threshold           = 1
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"

  alarm_actions = [aws_sns_topic.lambda_error.arn]
  ok_actions    = [aws_sns_topic.lambda_error.arn]
}

resource "aws_cloudwatch_metric_alarm" "initialize_error" {
  alarm_name          = "jishinranking-initialize-error"
  alarm_description   = "Lambda jishinranking-initialize でエラーが発生しました"
  namespace           = "AWS/Lambda"
  metric_name         = "Errors"
  dimensions = {
    FunctionName = aws_lambda_function.initialize.function_name
  }

  statistic           = "Sum"
  period              = 60
  evaluation_periods  = 1
  threshold           = 1
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"

  alarm_actions = [aws_sns_topic.lambda_error.arn]
  ok_actions    = [aws_sns_topic.lambda_error.arn]
}
