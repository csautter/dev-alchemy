terraform {
  required_version = ">= 1.8.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 4.57.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = ">= 3.7.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.7.2"
    }
  }
}
