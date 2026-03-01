output "azurerm_storage_account_name" {
  description = "The Azure Storage Account for Terraform state backend"
  value       = azurerm_storage_account.this.name
}

output "azurerm_storage_container_name" {
  description = "The Azure Storage Container for Terraform state backend"
  value       = azurerm_storage_container.this.name
}
