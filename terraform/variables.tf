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
  default     = "production"
}

variable "rssfeed_url" {
  description = "The URL of the RSS feed"
  type        = string

  validation {
    condition     = can(regex("^https?://", var.rssfeed_url))
    error_message = "The RSS feed URL must start with http:// or https://"
  }
}
