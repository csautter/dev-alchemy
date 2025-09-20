#!/bin/bash

VM_NAME="macos-ansible-test"
ISO_PATH="./vendor/macos/macos_installer_Sequoia.iso"
DISK_SIZE="64G"

if utmctl vm list | grep -q "$VM_NAME"; then
  echo "Removing existing VM $VM_NAME"
  utmctl vm delete "$VM_NAME"
fi

echo "Creating VM $VM_NAME"
utmctl vm create --name "$VM_NAME" --cpu 4 --memory 8192 --disk "$DISK_SIZE"

echo "Attaching ISO installer"
utmctl vm attach-iso "$VM_NAME" "$ISO_PATH"

echo "Starting VM"
utmctl vm start "$VM_NAME"

echo "VM created and started. Please complete macOS installation manually."
