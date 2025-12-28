import azure.functions as func
import json
import logging
import requests

from azure.identity import DefaultAzureCredential
from azure.keyvault.secrets import SecretClient

app = func.FunctionApp()


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
    kv_client = SecretClient(
        vault_url="https://gh-runner-kv.vault.azure.net", credential=credential
    )

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

    token = response.json()["token"]

    # ⚠️ TEMP: return token for testing
    return func.HttpResponse(
        json.dumps({"message": "Runner token created", "token": token}),
        mimetype="application/json",
        status_code=200,
    )
