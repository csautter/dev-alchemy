resource "random_integer" "suffix" {
  #min = 10000
  #max = 99999
  min = 18632
  max = 18632
}

resource "azurerm_resource_group" "gh_runner_manager" {
  name     = "gh-runner-manager"
  location = "francecentral"
}

resource "azurerm_key_vault" "gh_runner_kv" {
  name                       = "gh-runner-kv-b-${random_integer.suffix.result}"
  location                   = azurerm_resource_group.gh_runner_manager.location
  resource_group_name        = azurerm_resource_group.gh_runner_manager.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "standard"
  rbac_authorization_enabled = true
  purge_protection_enabled   = false
  soft_delete_retention_days = 7

  lifecycle {
    prevent_destroy = true
  }
}

data "azurerm_client_config" "current" {}

resource "azurerm_role_assignment" "keyvault_secrets_officer" {
  scope                = azurerm_key_vault.gh_runner_kv.id
  role_definition_name = "Key Vault Secrets Officer"
  principal_id         = data.azurerm_client_config.current.object_id
}
resource "azurerm_storage_account" "gh_runner_storage" {
  name                            = "ghrunnerstorage${random_integer.suffix.result}"
  resource_group_name             = azurerm_resource_group.gh_runner_manager.name
  location                        = azurerm_resource_group.gh_runner_manager.location
  account_tier                    = "Standard"
  account_replication_type        = "LRS"
  allow_nested_items_to_be_public = false

  lifecycle {
    prevent_destroy = true
  }
}

ephemeral "azurerm_key_vault_secret" "github_runner_pat" {
  name         = "github-runner-pat"
  key_vault_id = azurerm_key_vault.gh_runner_kv.id
}

ephemeral "azurerm_key_vault_secret" "github-runner-vm-admin-pw" {
  name         = "github-runner-vm-admin-pw"
  key_vault_id = azurerm_key_vault.gh_runner_kv.id
}

resource "azurerm_linux_function_app" "gh_runner_func_app" {
  name                                           = "gh-runner-func-app-${random_integer.suffix.result}"
  resource_group_name                            = azurerm_resource_group.gh_runner_manager.name
  location                                       = azurerm_resource_group.gh_runner_manager.location
  storage_account_name                           = azurerm_storage_account.gh_runner_storage.name
  storage_account_access_key                     = azurerm_storage_account.gh_runner_storage.primary_access_key
  service_plan_id                                = azurerm_service_plan.gh_runner_func_plan.id
  builtin_logging_enabled                        = false
  ftp_publish_basic_authentication_enabled       = false
  webdeploy_publish_basic_authentication_enabled = false
  site_config {
    http2_enabled                          = true
    ftps_state                             = "FtpsOnly"
    application_insights_connection_string = azurerm_application_insights.gh_runner_func_app.connection_string
    application_stack {
      python_version = "3.11"
    }
  }

  app_settings = {
    FUNCTIONS_WORKER_RUNTIME = "python"
    VAULT_URL                = azurerm_key_vault.gh_runner_kv.vault_uri
    SUBSCRIPTION_ID          = data.azurerm_client_config.current.subscription_id
    LOCATION                 = "eastus"
    RESOURCE_GROUP           = "gh-runner-tmp-rg"
    VM_NAME                  = "gh-runner-vm"
    VM_SIZE                  = "Standard_D2s_v3"
    ADMIN_USERNAME           = "azureuser"
    CUSTOM_IMAGE_ID          = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/resourceGroups/gh-actions-images/providers/Microsoft.Compute/images/Win2022GHAzureRunnerImage"
  }

  identity {
    type = "SystemAssigned"
  }

  tags = {
    "hidden-link: /app-insights-resource-id" = azurerm_application_insights.gh_runner_func_app.id
  }
}

resource "azurerm_service_plan" "gh_runner_func_plan" {
  name                     = "EastUSLinuxConsumption"
  resource_group_name      = azurerm_resource_group.gh_runner_manager.name
  location                 = azurerm_resource_group.gh_runner_manager.location
  os_type                  = "Linux"
  sku_name                 = "Y1"
  per_site_scaling_enabled = false
  lifecycle {
    ignore_changes = [worker_count]
  }
}

resource "azurerm_role_assignment" "keyvault_secrets_user" {
  scope                = azurerm_key_vault.gh_runner_kv.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_linux_function_app.gh_runner_func_app.identity[0].principal_id
}

resource "azurerm_role_assignment" "function_contributor" {
  scope                = data.azurerm_subscription.current.id
  role_definition_name = "Contributor"
  principal_id         = azurerm_linux_function_app.gh_runner_func_app.identity[0].principal_id
}

resource "azurerm_application_insights" "gh_runner_func_app" {
  name                = "gh-runner-func-app-${random_integer.suffix.result}"
  location            = azurerm_resource_group.gh_runner_manager.location
  resource_group_name = azurerm_resource_group.gh_runner_manager.name
  application_type    = "web"
}

data "azurerm_subscription" "current" {}

data "azurerm_role_definition" "monitoring_contributor" {
  name  = "Monitoring Contributor"
  scope = data.azurerm_subscription.current.id
}

data "azurerm_role_definition" "monitoring_reader" {
  name  = "Monitoring Reader"
  scope = data.azurerm_subscription.current.id
}

resource "azurerm_monitor_action_group" "smart_detection" {
  name                = "Application Insights Smart Detection"
  resource_group_name = azurerm_resource_group.gh_runner_manager.name
  short_name          = "SmartDetect"
  arm_role_receiver {
    name                    = "Monitoring Contributor"
    role_id                 = basename(data.azurerm_role_definition.monitoring_contributor.role_definition_id)
    use_common_alert_schema = true
  }
  arm_role_receiver {
    name                    = "Monitoring Reader"
    role_id                 = basename(data.azurerm_role_definition.monitoring_reader.role_definition_id)
    use_common_alert_schema = true
  }
}
