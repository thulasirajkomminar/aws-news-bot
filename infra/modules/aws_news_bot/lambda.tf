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
  version = "2.3.1"

  name          = local.name
  architecture  = "arm64"
  description   = "AWS News Bot Lambda"
  handler       = "bootstrap"
  log_retention = 3
  runtime       = "provided.al2"
  timeout       = 300
  tags          = local.tags

  environment = {
    BLUESKY_HANDLE        = var.bluesky_handle
    BLUESKY_PASSWORD_PATH = var.bluesky_password_path
    DYNAMODB_TABLE_NAME   = module.table.id
    NEWSBLOG_RSSFEED_URL  = var.newsblog_rssfeed_url
    WHATSNEW_RSSFEED_URL  = var.whatsnew_rssfeed_url
  }

  execution_role = {
    policy = data.aws_iam_policy_document.lambda_policy.json
  }
}

resource "aws_cloudwatch_event_rule" "default" {
  count = var.environment == "prd" ? 1 : 0

  name                = local.name
  description         = "Run ${local.name} every 1 hour"
  schedule_expression = "cron(5/60 * ? * * *)"
  state               = var.environment == "prd" ? "ENABLED" : "DISABLED"
}

resource "aws_cloudwatch_event_target" "default" {
  count = var.environment == "prd" ? 1 : 0

  rule      = aws_cloudwatch_event_rule.default[0].name
  target_id = local.name
  arn       = module.lambda.arn
}

resource "aws_lambda_permission" "default" {
  count = var.environment == "prd" ? 1 : 0

  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = module.lambda.name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.default[0].arn
}
