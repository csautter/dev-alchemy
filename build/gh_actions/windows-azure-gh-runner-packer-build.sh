#!/bin/bash
set -e

subscription_id="$(packer inspect -var-file=windows-azure-gh-runner-secrets.pkrvars.hcl windows-azure-gh-runner.pkr.hcl | grep subscription_id | cut -d'"' -f2)"
image_resource_group="gh-actions-images"
az group create --name "$image_resource_group" --location eastus --subscription "$subscription_id"
packer build -var-file=windows-azure-gh-runner-secrets.pkrvars.hcl -var="image_resource_group=$image_resource_group" windows-azure-gh-runner.pkr.hcl