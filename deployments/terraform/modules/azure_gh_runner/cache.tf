# Storage account for caching Windows ISOs and other build artifacts
resource "azurerm_resource_group" "cache_storage" {
  name     = "gh-runner-storage-rg"
  location = var.runner_location
}

resource "azurerm_storage_account" "cache" {
  name                            = substr(replace("ghrunner${data.azurerm_client_config.current.subscription_id}", "-", ""), 0, 24)
  resource_group_name             = azurerm_resource_group.cache_storage.name
  location                        = azurerm_resource_group.cache_storage.location
  account_tier                    = "Standard"
  account_replication_type        = "LRS"
  allow_nested_items_to_be_public = false
  min_tls_version                 = "TLS1_2"

  blob_properties {
    versioning_enabled = false
  }

  lifecycle {
    prevent_destroy = true
  }
}

# General-purpose build cache container for ISOs, toolchain archives, and other
# large build dependencies. Use this container for any new cached artifacts.
resource "azurerm_storage_container" "build_cache" {
  name                  = "build-cache"
  storage_account_id    = azurerm_storage_account.cache.id
  container_access_type = "private"

  lifecycle {
    prevent_destroy = true
  }
}

# Grant the GitHub Actions service principal permission to read the resource group
resource "azurerm_role_assignment" "gh_actions_cache_rg_reader" {
  scope                = azurerm_resource_group.cache_storage.id
  role_definition_name = "Reader"
  principal_id         = azuread_service_principal.gh_actions_runner_broker.object_id
}

# Grant the GitHub Actions service principal permission to read and upload blobs to the cache storage account
resource "azurerm_role_assignment" "gh_actions_cache_blob_contributor" {
  scope                = azurerm_storage_account.cache.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azuread_service_principal.gh_actions_runner_broker.object_id
}

# Grant the current user (deploying Terraform) permission to upload blobs to the cache storage account
resource "azurerm_role_assignment" "current_user_cache_blob_contributor" {
  scope                = azurerm_storage_account.cache.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = data.azurerm_client_config.current.object_id
}
