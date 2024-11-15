data "aws_iam_policy_document" "lambda_policy" {
  statement {
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:UpdateItem",
      "dynamodb:DeleteItem",
      "ssm:Describe*",
      "ssm:Get*",
      "ssm:List*",
    ]
    resources = ["*"]
  }
}

module "lambda" {
  source  = "schubergphilis/mcaf-lambda/aws"
  version = "1.4.1"

  name          = local.name
  description   = "AWS News Update Lambda"
  runtime       = "provided.al2"
  handler       = "bootstrap"
  architecture  = "arm64"
  create_policy = true
  log_retention = 3
  policy        = data.aws_iam_policy_document.lambda_policy.json
  timeout       = 300
  tags          = local.tags

  environment = {
    BLUESKY_HANDLE        = var.bluesky_handle
    BLUESKY_PASSWORD_PATH = var.bluesky_password_path
    DYNAMODB_TABLE_NAME   = module.table.id
    NEWSBLOG_RSSFEED_URL  = var.newsblog_rssfeed_url
    WHATSNEW_RSSFEED_URL  = var.whatsnew_rssfeed_url
  }
}

resource "aws_cloudwatch_event_rule" "default" {
  name                = local.name
  description         = "Run ${local.name} every 1 hour"
  schedule_expression = "cron(5/60 * ? * * *)"
  state               = "ENABLED"
}

resource "aws_cloudwatch_event_target" "default" {
  rule      = aws_cloudwatch_event_rule.default.name
  target_id = local.name
  arn       = module.lambda.arn
}

resource "aws_lambda_permission" "default" {
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = module.lambda.name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.default.arn
}
