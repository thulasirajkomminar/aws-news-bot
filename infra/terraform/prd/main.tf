module "aws_news_bot" {
  source = "../../modules/aws_news_bot"

  bluesky_handle        = var.bluesky_handle
  bluesky_password_path = var.bluesky_password_path
  environment           = var.environment
  newsblog_rssfeed_url  = var.newsblog_rssfeed_url
  whatsnew_rssfeed_url  = var.whatsnew_rssfeed_url
}
