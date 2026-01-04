import base64
import azure.functions as func
import json
import logging
import requests
import os

from azure.identity import DefaultAzureCredential
from azure.keyvault.secrets import SecretClient
from azure.mgmt.compute import ComputeManagementClient
from azure.mgmt.resource import ResourceManagementClient

app = func.FunctionApp()

POWERSHELL_TEMPLATE = r"""
<powershell>
$ErrorActionPreference = "Stop"

# write a file that logs if this file was executed
New-Item -Path "C:\" -Name "execution.log" -ItemType "file" -Force
Add-Content -Path "C:\execution.log" -Value "cloud-init-combined.ps1 executed on $(Get-Date)"

# Install Hyper-V
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All -NoRestart
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V-Management-PowerShell -All -NoRestart
# Optionally, restart if required
if ((Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V).State -ne 'Enabled') {
  Write-Host 'Restart required to complete Hyper-V installation.'
  Restart-Computer -Force
  exit
}

$RunnerToken = "__RUNNER_TOKEN__"
$RepoUrl     = "__REPO_URL__"
$RunnerName  = $env:COMPUTERNAME
$RunnerDir   = "C:\actions-runner"

New-Item -ItemType Directory -Force -Path $RunnerDir
Set-Location $RunnerDir

.\config.cmd `
  --url $RepoUrl `
  --token $RunnerToken `
  --name $RunnerName `
  --labels windows,azure,nested `
  --unattended `
  --ephemeral `
  --runasservice
</powershell>
"""


@app.route(route="request_runner", auth_level=func.AuthLevel.FUNCTION)
def request_runner(req: func.HttpRequest) -> func.HttpResponse:
    logging.info("Request received")

    try:
        body = req.get_json()
        repo = body["repo"]  # org/repo
    except Exception:
        return func.HttpResponse(
            'Invalid JSON body. Expected: { "repo": "org/repo" }', status_code=400
        )

    # 1. Get PAT from Key Vault
    credential = DefaultAzureCredential()
    vault_url = os.environ["VAULT_URL"]
    kv_client = SecretClient(vault_url=vault_url, credential=credential)

    pat = kv_client.get_secret("github-runner-pat").value

    # 2. Call GitHub API
    url = f"https://api.github.com/repos/{repo}/actions/runners/registration-token"

    response = requests.post(
        url,
        headers={
            "Authorization": f"Bearer {pat}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        },
        timeout=10,
    )

    if response.status_code != 201:
        return func.HttpResponse(
            f"GitHub API error: {response.status_code} {response.text}", status_code=500
        )

    runner_token = response.json()["token"]

    # ⚠️ TEMP: return token for testing
    # return func.HttpResponse(
    #    json.dumps({"message": "Runner token created", "token": token}),
    #    mimetype="application/json",
    #    status_code=200,
    # )
    script = POWERSHELL_TEMPLATE.replace("__RUNNER_TOKEN__", runner_token).replace(
        "__REPO_URL__", f"https://github.com/{repo}"
    )

    custom_data = base64.b64encode(script.encode("utf-8")).decode("utf-8")

    # Create VM
    compute = ComputeManagementClient(credential, os.environ["SUBSCRIPTION_ID"])
    resource_client = ResourceManagementClient(
        credential, os.environ["SUBSCRIPTION_ID"]
    )

    resource_client.resource_groups.create_or_update(
        os.environ["RESOURCE_GROUP"], {"location": os.environ["LOCATION"]}
    )

    # Support custom image via environment variable (expects ARM resource ID)
    custom_image_id = os.environ.get("CUSTOM_IMAGE_ID")
    custom_image_location = os.environ.get("CUSTOM_IMAGE_LOCATION")
    if custom_image_id and custom_image_location:
        image_reference = {"id": custom_image_id}
    else:
        # throw error if no custom image provided
        return func.HttpResponse(
            "CUSTOM_IMAGE_ID environment variable not set", status_code=500
        )

    try:
        admin_password = kv_client.get_secret("github-runner-vm-admin-pw").value
    except Exception as e:
        return func.HttpResponse(
            f"Error retrieving VM admin password from Key Vault: {str(e)}",
            status_code=500,
        )

    try:
        vm_params = {
            "location": os.environ["LOCATION"],
            "hardware_profile": {"vm_size": os.environ["VM_SIZE"]},
            "storage_profile": {"image_reference": image_reference},
            "os_profile": {
                "computer_name": os.environ["VM_NAME"],
                "admin_username": os.environ["ADMIN_USERNAME"],
                "admin_password": admin_password,
                "custom_data": custom_data,
            },
            "network_profile": {
                # keep simple for now – NIC creation omitted for brevity
            },
        }

        compute.virtual_machines.begin_create_or_update(
            os.environ["RESOURCE_GROUP"], os.environ["VM_NAME"], vm_params
        )
    except Exception as e:
        return func.HttpResponse(
            f"Error creating VM: {str(e)}",
            status_code=500,
        )

    return func.HttpResponse(
        json.dumps(
            {"message": "Runner VM creation started", "vm": os.environ["VM_NAME"]}
        ),
        mimetype="application/json",
        status_code=202,
    )
