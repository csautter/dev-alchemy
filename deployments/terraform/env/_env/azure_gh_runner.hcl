locals {
  env_local = try(read_terragrunt_config(find_in_parent_folders("env_local.hcl")), { locals = {} })
  env       = read_terragrunt_config(find_in_parent_folders("env.hcl"))
  config = merge(
    local.env.locals,
    local.env_local.locals
  )
}

dependency "azure_state_backend" {
  config_path = "${get_terragrunt_dir()}/../../${local.config.env}/azure_state_backend"
}

terraform {
  source = "${get_terragrunt_dir()}/../../../modules//azure_gh_runner"
}

inputs = {

}