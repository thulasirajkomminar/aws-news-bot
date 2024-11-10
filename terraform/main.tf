locals {
  name = "AWS-News-Update-${data.aws_region.current.name}"

  tags = {
    Stack       = "AWS-News-Update"
    Environment = var.environment
    Project     = "AWS-News"
    Owner       = "Platform-Team"
    ManagedBy   = "Terraform"
  }
}

data "aws_region" "current" {}
