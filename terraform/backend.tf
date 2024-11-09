terraform {
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "komminarlabs"

    workspaces {
      name = "aws-news-update"
    }
  }
}
