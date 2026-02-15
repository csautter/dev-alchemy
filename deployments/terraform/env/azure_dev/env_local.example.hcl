# copy this file to env_local.hcl and update the values accordingly
locals {
  subscription_id              = "<your_subscription_id>"
  tenant_id                    = "<your_tenant_id>"
  state_backend                = "azure"
  state_storage_account_name   = "<your_desired_storage_account_name>"
  state_storage_container_name = "<your_desired_storage_container_name>"
}