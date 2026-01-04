resource "azurerm_resource_group" "this" {
  name     = "azure-github-runner-rg-${var.env}"
  location = data.azurerm_location.east_us.location
  tags = {
    shortname = "azure-github-runner"
    env       = var.env
  }
}
