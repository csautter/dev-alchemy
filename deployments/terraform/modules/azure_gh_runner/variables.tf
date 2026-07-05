variable "env" {
  description = "The environment name"
  type        = string
}

variable "runner_location" {
  description = "Azure region for the runner resources"
  type        = string
  default     = "eastus2"
}

variable "github_repository" {
  description = "GitHub repository allowed to authenticate with Azure OIDC, in owner/repo form."
  type        = string
  default     = "csautter/sailwright"

  validation {
    condition     = can(regex("^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$", var.github_repository))
    error_message = "github_repository must be in owner/repo form."
  }
}

variable "allowed_user_object_ids" {
  description = "Optional list of Entra user object IDs allowed to call broker endpoints directly."
  type        = list(string)
  default     = []
}
