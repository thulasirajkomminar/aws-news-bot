variable "bluesky_handle" {
  description = "The handle to the Aurora cluster"
  type        = string
}

variable "bluesky_password_path" {
  description = "The path to the password in SSM"
  type        = string
}

variable "rssfeed_url" {
  description = "The URL of the RSS feed"
  type        = string
}
