# resource "aws_scheduler_schedule" "cron" {
#   name       = "cron"
#   schedule_expression = "rate(10 minutes)"

#   target {
#     arn      = "arn:aws:scheduler:::aws-sdk:sqs:sendMessage"
#     role_arn = aws_iam_role.example.arn

#     input = ""
#   }
# }

resource "aws_lambda_function" "poller" {
  function_name = "poller"
  role          = aws_iam_role.lambda_poller_role.arn
  filename      = "../src/default.zip"
  runtime       = "go1.x"
  handler       = "poller"

  tags       = local.tags
  depends_on = [aws_iam_role_policy_attachment.lambda_policy_poller_attachment]

  lifecycle {
    ignore_changes = [
      filename,
    ]
  }
}

resource "aws_lambda_function" "notifier" {
  function_name = "notifier"
  role          = aws_iam_role.lambda_role.arn
  filename      = "../src/default.zip"
  runtime       = "go1.x"
  handler       = "notifier"

  tags       = local.tags
  depends_on = [aws_iam_role_policy_attachment.lambda_policy_attachment]

  lifecycle {
    ignore_changes = [
      filename,
    ]
  }
}

resource "aws_dynamodb_table" "state-store" {
  name           = "state-store"
  billing_mode   = "PROVISIONED"
  read_capacity  = 1
  write_capacity = 1
  hash_key       = "LocationId"

  attribute {
    name = "LocationId"
    type = "S"
  }

  tags = local.tags
}