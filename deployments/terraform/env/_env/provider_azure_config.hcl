locals {
  env_local = try(read_terragrunt_config(find_in_parent_folders("env_local.hcl")), { locals = {} })
  env       = read_terragrunt_config(find_in_parent_folders("env.hcl"))
  config = merge(
    local.env.locals,
    local.env_local.locals
  )
}

generate "azurerm" {
  path      = "provider_azurem.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<-EOF
     provider "azurerm" {
        resource_provider_registrations = "none" # This is only required when the User, Service Principal, or Identity running Terraform lacks the permissions to register Azure Resource Providers.
        features {}
        subscription_id = "${local.config.subscription_id}"
     }
  EOF
}
