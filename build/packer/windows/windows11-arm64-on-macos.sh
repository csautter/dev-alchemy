#!/usr/bin/env bash

SCRIPT_DIR=$(cd $(dirname $0); pwd -P)
PROJECT_ROOT=$(cd ${SCRIPT_DIR}/../../..; pwd -P)

cd ${PROJECT_ROOT}

# download the Windows 11 ARM64 ISO if not already present
if [ ! -d ./vendor/windows ]; then
  mkdir -p ./vendor/windows
fi
if [ ! -f ./vendor/windows/win11_25H2_english_arm64.iso ]; then
  echo "Downloading Windows 11 ARM64 ISO"
  cd ./scripts/macos/
  source .venv/bin/activate
  python playwright_win11_iso.py --arm
  cd ./vendor/windows/
  curl --progress-bar -o win11_25h2_english_arm64.iso $(cat ./win11_arm_iso_url.txt)
  cd ${PROJECT_ROOT}
else
  echo "Windows 11 ARM64 ISO already exists, skipping download"
fi

# download the qemu-uefi files if not already present
bash scripts/macos/download-arm64-uefi.sh

# builds the autounattend ISO with the current autounattend.xml file
bash scripts/macos/create-win11-autounattend-iso.sh

# creates the qcow2 disk image and overwrites it if it already exists
bash scripts/macos/create-qemu-qcow2-disk.sh

PACKER_LOG=1 packer build -var "iso_url=./vendor/windows/win11_25H2_english_arm64.iso" build/packer/windows/windows11-arm64-on-macos.pkr.hcl