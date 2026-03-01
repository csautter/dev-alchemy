resource "azurerm_resource_group" "state" {
  name     = "terraform-state-rg-${var.env}"
  location = data.azurerm_location.this.location
  tags = {
    shortname = "terraform-state"
    env       = var.env
  }

  lifecycle {
    prevent_destroy = true
  }
}

data "azurerm_location" "this" {
  location = var.location
}
