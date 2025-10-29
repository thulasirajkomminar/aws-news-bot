terraform {
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "thulasirajkomminar"

    workspaces {
      name = "aws-news-bot-dev"
    }
  }
}
