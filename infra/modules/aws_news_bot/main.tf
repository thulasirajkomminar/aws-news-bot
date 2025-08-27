locals {
  name = "AWS-News-Bot-${var.environment}-${data.aws_region.current.region}"

  tags = {
    Stack       = "AWS-News-Bot"
    Environment = var.environment
    Project     = "AWS-News"
    Owner       = "Platform-Team"
    ManagedBy   = "Terraform"
  }
}

data "aws_region" "current" {}
