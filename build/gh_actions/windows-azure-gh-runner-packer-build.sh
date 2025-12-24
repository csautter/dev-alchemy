#!/bin/bash
set -e

packer build -var-file=build/gh_actions/windows-azure-gh-runner-secrets.pkrvars.hcl build/gh_actions/windows-azure-gh-runner.pkr.hcl