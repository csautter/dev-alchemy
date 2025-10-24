#!/usr/bin/env bash

set -ex
# This script creates a Windows 11 ARM64 UTM VM on macOS using the specified configuration.

script_dir=$(
	cd "$(dirname "$0")"
	pwd
)
project_root=$(
	cd "${script_dir}/../../.."
	pwd
)

utm_vm_dir="/Users/$(whoami)/Library/Containers/com.utmapp.UTM/Data/Documents"

if
	[ -d "$utm_vm_dir" ] &&
		[ ! -d "$utm_vm_dir/Windows11-ARM64-dev-alchemy.utm/Data" ]
then
	mkdir -p "$utm_vm_dir/Windows11-ARM64-dev-alchemy.utm/Data"
fi

if [ ! -f "$project_root/vendor/windows/qemu-windows11-arm64.qcow2" ]; then
	echo "The required qcow2 image for Windows 11 ARM64 is missing. Please run the packer build script first."
	exit 1
fi

# Generate a random MAC address
generate_mac_address() {
	hexchars="0123456789ABCDEF"
	echo "A6:$(for i in {1..5}; do
		echo -n ${hexchars:$((RANDOM % 16)):1}${hexchars:$((RANDOM % 16)):1}
		[ $i -lt 5 ] && echo -n ":"
	done)"
}

mac_address=$(generate_mac_address)
echo "Generated random MAC address: $mac_address"

# Generate a random UUID with the same schema as BBEF1D33-5B60-40A9-B2DF-57E919EEF921
generate_uuid() {
	# Use uuidgen and convert to uppercase
	uuid=$(uuidgen | tr 'a-f' 'A-F')
	echo "$uuid"
}

QCOW_IMAGE="windows11-arm64.qcow2" \
	VM_NAME="Windows11-ARM64-dev-alchemy" \
	MAC_ADDRESS="$mac_address" \
	UUID="$(generate_uuid)" \
	UUID_CD="$(generate_uuid)" \
	UUID_DISK="$(generate_uuid)" \
	envsubst <"$project_root/deployments/utm/windows11-arm64/config.plist" >"$utm_vm_dir/Windows11-ARM64-dev-alchemy.utm/config.plist"

if [ ! -f "$utm_vm_dir/Windows11-ARM64-dev-alchemy.utm/Data/windows11-arm64.qcow2" ]; then
	echo "Copying the qcow2 image to UTM VM directory..."
	cp "$project_root/vendor/windows/qemu-windows11-arm64.qcow2" "$utm_vm_dir/Windows11-ARM64-dev-alchemy.utm/Data/windows11-arm64.qcow2"
else
	echo "The qcow2 image already exists in the UTM VM directory. Skipping copy."
fi
echo "Windows 11 ARM64 UTM VM setup is complete. You can now open UTM and start the VM."
