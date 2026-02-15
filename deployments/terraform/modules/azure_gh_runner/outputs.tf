output "azure_tenant_id" {
  value       = data.azurerm_client_config.current.tenant_id
  description = "The Azure tenant ID"
}

output "azure_ad_app_client_id" {
  value       = azuread_application.gh_actions_runner_broker.client_id
  description = "The Azure AD Application (client) ID for the GitHub Actions OIDC federated identity"
}

output "azure_subscription_id" {
  value       = data.azurerm_client_config.current.subscription_id
  description = "The Azure subscription ID"
}

output "function_app_default_hostname" {
  value       = azurerm_linux_function_app.gh_runner_func_app.default_hostname
  description = "The default hostname of the GitHub Actions runner function app"
}

output "function_app_name" {
  value       = azurerm_linux_function_app.gh_runner_func_app.name
  description = "The name of the GitHub Actions runner function app"
}
