````bash
# get gh runner registration token
TOKEN=$(curl -sX POST -H "Authorization: Bearer ${{ secrets.GITHUB_TOKEN }}" \
-H "Accept: application/vnd.github+json" \
https://api.github.com/repos/${{ github.repository }}/actions/runners/registration-token | jq -r .token)

# or set token manually for testing
TOKEN=<TOKEN_VALUE>

# create combined cloud-init script with token and repo url
echo "\$env:TOKEN=$TOKEN" >> gh-runner.txt.tmp
echo "\$env:REPO_URL=https://github.com/csautter/dev-alchemy" >> gh-runner.txt.tmp
cat gh-runner.txt.tmp scripts/gh_actions/cloud-init.ps1 > scripts/gh_actions/cloud-init-combined.ps1

# create vm with cloud-init script to install gh runner and register it
resource_group="gh_actions"
password="P@ssw0rd1234!" # use a strong password and store it securely

machine_type="Standard_D2s_v3" # needs to support nested virtualization

az group create \
--name $resource_group \
--location "East US"

image="/subscriptions/<subscription_id>/resourceGroups/packerResourceGroup/providers/Microsoft.Compute/images/Win2022GHAzureRunnerImage"

az vm create \
--resource-group $resource_group \
--name win-runner \
--image $image \
--size $machine_type \
--admin-username azureuser \
--admin-password "$password" \
--custom-data scripts/gh_actions/cloud-init-combined.ps1

# optional parameters for spot instance
--priority Spot \
--eviction-policy Delete \
--nic-delete-option delete \
--os-disk-delete-option delete \

az vm delete --resource-group $resource_group --name win-runner --yes
# clean up resource group
az group delete --name $resource_group --yes --no-wait
````