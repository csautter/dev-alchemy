#!/usr/bin/env bash

SCRIPT_DIR=$(cd $(dirname $0); pwd -P)
PROJECT_ROOT=$(cd ${SCRIPT_DIR}/../../..; pwd -P)

cd ${PROJECT_ROOT}
bash scripts/macos/create-win11-autounattend-iso.sh

bash scripts/macos/create-qemu-qcow2-disk.sh

PACKER_LOG=1 packer build -var "iso_url=./vendor/windows/win11_25H2_english_arm64.iso" build/packer/windows/windows11-arm64-on-macos.pkr.hcl