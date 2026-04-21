#!/bin/bash
set -e

packer init windows-azure-gh-runner.pkr.hcl
subscription_id="$(packer inspect -var-file=windows-azure-gh-runner-secrets.pkrvars.hcl windows-azure-gh-runner.pkr.hcl | grep subscription_id | cut -d'"' -f2)"
location="eastus2"
image_resource_group="gh-actions-images-${location// /-}"
az group create --name "$image_resource_group" --location "$location" --subscription "$subscription_id"
packer build -var-file=windows-azure-gh-runner-secrets.pkrvars.hcl -var="virtualization_flavor=hyperv" -var="image_resource_group=$image_resource_group" -var="location=$location" -force windows-azure-gh-runner.pkr.hcl
packer build -var-file=windows-azure-gh-runner-secrets.pkrvars.hcl -var="virtualization_flavor=virtualbox" -var="image_resource_group=$image_resource_group" -var="location=$location" -force windows-azure-gh-runner.pkr.hcl