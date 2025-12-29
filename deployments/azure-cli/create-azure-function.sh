#!/bin/bash
set -e

resource_group="gh-runner-manager"
region="eastus"
random=$(shuf -i 10000-99999 -n 1)
random=18632
keyvault_name="gh-runner-kv-$random"
subscription_id=$(az account show --query id -o tsv)

az group create --name $resource_group --location $region

# create key vault
az keyvault create \
  --name $keyvault_name \
  --resource-group $resource_group \
  --location $region

az keyvault show --name $keyvault_name --query properties.enableRbacAuthorization --output tsv

scope=$(az keyvault show --name $keyvault_name --query id -o tsv | tr -d '\n')
scope="${scope#/}"
assignee="<your-identity-object-id-or-UPN>"
az role assignment create \
  --role "Key Vault Secrets Officer" \
  --subscription "$subscription_id" \
  --scope "$scope" \
  --assignee "$assignee"

az keyvault secret set \
  --vault-name $keyvault_name \
  --name github-runner-pat \
  --value "<YOUR_PAT>"

# create storage account for function app
storage_account_name=ghrunnerstorage$random

# enable resource provider for storage accounts
az provider register --namespace Microsoft.Storage

az storage account create \
  --name $storage_account_name \
  --location $region \
  --resource-group $resource_group \
  --sku Standard_LRS

function_app_name=gh-runner-func-app-$random

# enable resource provider for function apps
az provider register --namespace Microsoft.Web

az functionapp create \
  --name $function_app_name \
  --resource-group $resource_group \
  --consumption-plan-location $region \
  --runtime python \
  --runtime-version 3.13 \
  --os-type Linux \
  --functions-version 4 \
  --storage-account $storage_account_name

# enable managed identity for the function app

az functionapp identity assign \
  --name $function_app_name \
  --resource-group $resource_group

# get the principal ID of the function app's managed identity
function_app_principal_id=$(az functionapp identity show \
  --name $function_app_name \
  --resource-group $resource_group \
  --query principalId \
  --output tsv
)

# grant key vault access to the function app's managed identity
# todo -> fix issues with role assignment
az role assignment create \
  --role "Key Vault Secrets User" \
  --assignee "$function_app_principal_id" \
  --scope "/$scope"

# test local
func start

# deploy the function app code
func azure functionapp publish $function_app_name

# test the function app
az functionapp function keys list \
  --function-name request_runner \
  --name $function_app_name \
  --resource-group $resource_group

# test with curl
curl -X POST \
  "https://gh-runner-func-app.azurewebsites.net/api/request_runner?code=<KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "repo": "ORG/REPO"
  }'

# expected response:
# {
#  "message": "Runner token created",
#  "token": "AAAA...."
#}
