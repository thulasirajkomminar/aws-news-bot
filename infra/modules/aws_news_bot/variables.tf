variable "bluesky_handle" {
  description = "The Bluesky social media handle for authentication"
  type        = string
}

variable "bluesky_password_path" {
  description = "The path to the password in SSM"
  type        = string
  sensitive   = true
}

variable "environment" {
  type        = string
  description = "Environment name for deployment"
}

variable "newsblog_rssfeed_url" {
  description = "The URL of the news blog RSS feed"
  type        = string
}

variable "whatsnew_rssfeed_url" {
  description = "The URL of the what's new RSS feed"
  type        = string
}
