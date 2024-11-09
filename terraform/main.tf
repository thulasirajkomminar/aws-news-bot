locals {
  name = "AWS-News-Update-${data.aws_region.current.name}"

  tags = {
    Stack       = "AWS-News-Update"
    Environment = "production"
  }
}

data "aws_region" "current" {}
