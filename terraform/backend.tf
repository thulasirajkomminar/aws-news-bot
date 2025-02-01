terraform {
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "KomminarLabs"

    workspaces {
      name = "aws-news"
    }
  }
}
