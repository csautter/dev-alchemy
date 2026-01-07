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
from azure.mgmt.network import NetworkManagementClient

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


@app.route(
    route="delete_resource_group",
    auth_level=func.AuthLevel.FUNCTION,
    methods=["DELETE"],
)
def delete_resource_group(req: func.HttpRequest) -> func.HttpResponse:
    logging.info("Delete resource group request received")
    try:
        credential = DefaultAzureCredential()
        subscription_id = os.environ["SUBSCRIPTION_ID"]
        # TODO: resource group name needs to follow a naming convention to avoid deleting unintended groups
        # TODO: create a check to validate the resource group is intended for deletion
        resource_group = os.environ["RESOURCE_GROUP"]
        resource_client = ResourceManagementClient(credential, subscription_id)
        # Delete the resource group and all resources within
        delete_async_op = resource_client.resource_groups.begin_delete(resource_group)
        delete_async_op.wait()
        return func.HttpResponse(
            json.dumps({"message": f"Resource group '{resource_group}' deleted."}),
            mimetype="application/json",
            status_code=200,
        )
    except Exception as e:
        return func.HttpResponse(
            f"Error deleting resource group: {str(e)}",
            status_code=500,
        )


@app.route(
    route="request_runner", auth_level=func.AuthLevel.ANONYMOUS, methods=["POST"]
)
def request_runner(req: func.HttpRequest) -> func.HttpResponse:
    # Case 1: OIDC identity
    principal = req.headers.get("X-MS-CLIENT-PRINCIPAL")

    if principal:
        claims = json.loads(base64.b64decode(principal))
        repo = claims.get("repository")
        ref = claims.get("ref")
        actor = claims.get("actor")
        # validate repo, ref, actor
        if (
            repo != "csautter/dev-alchemy"
            or not ref.startswith("refs/heads/")
            or actor != "csautter"
        ):
            return func.HttpResponse("Unauthorized", status_code=401)

        return handle_request_runner(req)

    # Case 2: Function key
    if req.route_params.get("code") or req.headers.get("x-functions-key"):
        return handle_request_runner(req)
    return func.HttpResponse("Unauthorized", status_code=401)


def handle_request_runner(req: func.HttpRequest) -> func.HttpResponse:
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
    if custom_image_id:
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
        # Create network resources
        network_client = NetworkManagementClient(
            credential, os.environ["SUBSCRIPTION_ID"]
        )

        vnet_name = os.environ.get("VNET_NAME", "gh-runner-vnet")
        subnet_name = os.environ.get("SUBNET_NAME", "default")
        ip_name = os.environ.get("IP_NAME", "gh-runner-ip")
        nic_name = os.environ.get("NIC_NAME", "gh-runner-nic")
        location = os.environ["LOCATION"]
        resource_group = os.environ["RESOURCE_GROUP"]

        # Create VNet if not exists
        vnet = network_client.virtual_networks.begin_create_or_update(
            resource_group,
            vnet_name,
            {
                "location": location,
                "address_space": {"address_prefixes": ["10.0.0.0/16"]},
            },
        ).result()

        # Create Subnet if not exists
        subnet = network_client.subnets.begin_create_or_update(
            resource_group,
            vnet_name,
            subnet_name,
            {"address_prefix": "10.0.0.0/24"},
        ).result()

        # Create Public IP
        public_ip = network_client.public_ip_addresses.begin_create_or_update(
            resource_group,
            ip_name,
            {
                "location": location,
                "public_ip_allocation_method": "Static",
                "sku": {"name": "Standard"},
            },
        ).result()

        # Create NSG (if not exists)
        nsg_name = os.environ.get("NSG_NAME", "gh-runner-nsg")
        nsg = network_client.network_security_groups.begin_create_or_update(
            resource_group,
            nsg_name,
            {
                "location": location,
            },
        ).result()

        # Add inbound rule for RDP (3389)
        network_client.security_rules.begin_create_or_update(
            resource_group,
            nsg_name,
            "Allow-RDP-Inbound",
            {
                "protocol": "Tcp",
                "source_port_range": "*",
                "destination_port_range": "3389",
                "source_address_prefix": "*",
                "destination_address_prefix": "*",
                "access": "Allow",
                "priority": 1000,
                "direction": "Inbound",
            },
        ).result()

        # Create NIC and associate NSG
        nic = network_client.network_interfaces.begin_create_or_update(
            resource_group,
            nic_name,
            {
                "location": location,
                "ip_configurations": [
                    {
                        "name": "ipconfig1",
                        "subnet": {"id": subnet.id},
                        "public_ip_address": {"id": public_ip.id},
                    }
                ],
                "network_security_group": {"id": nsg.id},
            },
        ).result()

        vm_params = {
            "location": location,
            "hardware_profile": {"vm_size": os.environ["VM_SIZE"]},
            "storage_profile": {"image_reference": image_reference},
            "os_profile": {
                "computer_name": os.environ["VM_NAME"],
                "admin_username": os.environ["ADMIN_USERNAME"],
                "admin_password": admin_password,
                "custom_data": custom_data,
            },
            "network_profile": {"network_interfaces": [{"id": nic.id}]},
        }

        compute.virtual_machines.begin_create_or_update(
            resource_group, os.environ["VM_NAME"], vm_params
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
