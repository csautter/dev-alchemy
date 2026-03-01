variable "env" {
  description = "The environment name"
  type        = string
}

variable "runner_location" {
  description = "Azure region for the runner resources"
  type        = string
  default     = "eastus2"
}

variable "allowed_user_object_ids" {
  description = "Optional list of Entra user object IDs allowed to call broker endpoints directly."
  type        = list(string)
  default     = []
}
