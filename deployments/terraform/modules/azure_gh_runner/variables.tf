variable "env" {
  description = "The environment name"
  type        = string
}

variable "runner_location" {
  description = "Azure region for the runner resources"
  type        = string
  default     = "eastus2"
}
