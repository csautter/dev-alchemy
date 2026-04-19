#!/usr/bin/env bash

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y ansible ca-certificates curl gpg ruby-full yq

. /etc/os-release

curl -fsSL https://apt.releases.hashicorp.com/gpg |
	gpg --dearmor --yes -o /usr/share/keyrings/hashicorp-archive-keyring.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com ${VERSION_CODENAME} main" \
	> /etc/apt/sources.list.d/hashicorp.list

apt-get update
apt-get install -y packer

go install github.com/securego/gosec/v2/cmd/gosec@v2.25.0

ansible --version
