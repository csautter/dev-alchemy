locals {
  env_local = try(read_terragrunt_config("${get_original_terragrunt_dir()}/../env_local.hcl"), { locals = {} })
  env       = read_terragrunt_config("${get_original_terragrunt_dir()}/../env.hcl")
  config = merge(
    local.env.locals,
    local.env_local.locals
  )
}

remote_state {
  backend      = "azurerm"
  disable_init = true

  generate = {
    path      = "backend_azurerm.tf"
    if_exists = "overwrite_terragrunt"
  }

  config = {
    use_cli              = true
    use_azuread_auth     = true
    tenant_id            = local.config.tenant_id
    storage_account_name = local.config.state_storage_account_name
    container_name       = local.config.state_storage_container_name
    key                  = "${local.config.env}.terraform.tfstate"
  }
}