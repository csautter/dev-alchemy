#!/bin/bash
set -e

# test local
func start

# deploy the function app code
function_app_name="gh-runner-func-app"
func azure functionapp publish $function_app_name

# test the function app
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
