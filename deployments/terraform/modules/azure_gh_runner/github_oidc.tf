resource "azuread_application" "gh_actions_runner_broker" {
  display_name = "gh-actions-runner-broker"
}

resource "azuread_service_principal" "gh_actions_runner_broker" {
  client_id = azuread_application.gh_actions_runner_broker.client_id
}

resource "azurerm_role_assignment" "gh_actions_runner_broker_contributor" {
  principal_id         = azuread_service_principal.gh_actions_runner_broker.object_id
  role_definition_name = "Contributor"
  scope                = azurerm_linux_function_app.gh_runner_func_app.id
}

resource "azuread_application_federated_identity_credential" "gh_actions_github_actions" {
  application_id = azuread_application.gh_actions_runner_broker.id
  display_name   = "github-actions"
  issuer         = "https://token.actions.githubusercontent.com"
  subject        = "repo:csautter/dev-alchemy:ref:refs/heads/main"
  audiences      = ["api://AzureADTokenExchange"]
}

# Allow GitHub Actions pull request workflows to access as well
resource "azuread_application_federated_identity_credential" "gh_actions_github_actions_pr" {
  application_id = azuread_application.gh_actions_runner_broker.id
  display_name   = "github-actions-pr"
  issuer         = "https://token.actions.githubusercontent.com"
  subject        = "repo:csautter/dev-alchemy:ref:refs/pull/*"
  audiences      = ["api://AzureADTokenExchange"]
}
