variable "bluesky_handle" {
  description = "The Bluesky social media handle for authentication"
  type        = string
  validation {
    condition     = can(regex("^[a-zA-Z0-9.-]+$", var.bluesky_handle))
    error_message = "The bluesky_handle must contain only alphanumeric characters, dots, and hyphens."
  }
}

variable "bluesky_password_path" {
  description = "The path to the password in SSM"
  type        = string
  sensitive   = true

  validation {
    condition     = can(regex("^/[a-zA-Z0-9-_/]+$", var.bluesky_password_path))
    error_message = "The SSM parameter path must start with / and contain only alphanumeric characters, hyphens, and underscores."
  }
}

variable "environment" {
  type        = string
  description = "Environment name for deployment"

  validation {
    condition     = contains(["dev", "prd"], var.environment)
    error_message = "Allowed values for environment are \"dev\", \"prd\"."
  }
}

variable "newsblog_rssfeed_url" {
  description = "The URL of the news blog RSS feed"
  type        = string

  validation {
    condition     = can(regex("^https?://", var.newsblog_rssfeed_url))
    error_message = "The news blog RSS feed URL must start with http:// or https://"
  }
}

variable "whatsnew_rssfeed_url" {
  description = "The URL of the what's new RSS feed"
  type        = string

  validation {
    condition     = can(regex("^https?://", var.whatsnew_rssfeed_url))
    error_message = "The what't new RSS feed URL must start with http:// or https://"
  }
}
