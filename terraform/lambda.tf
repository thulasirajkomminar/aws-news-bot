data "aws_iam_policy_document" "lambda_policy" {
  statement {
    actions = [
      "dynamodb:*",
      "ssm:Describe*",
      "ssm:Get*",
      "ssm:List*",
    ]
    resources = ["*"]
  }
}

module "lambda" {
  source  = "schubergphilis/mcaf-lambda/aws"
  version = "~> 1.4.1"

  name          = local.name
  description   = "AWS News Update Lambda"
  runtime       = "provided.al2"
  handler       = "bootstrap"
  architecture  = "arm64"
  create_policy = true
  log_retention = 3
  policy        = data.aws_iam_policy_document.lambda_policy.json
  tags          = local.tags

  environment = {
    BLUESKY_HANDLE        = var.bluesky_handle
    BLUESKY_PASSWORD_PATH = var.bluesky_password_path
    DYNAMODB_TABLE_NAME   = module.table.id
    RSSFEED_URL           = var.rssfeed_url
  }
}
