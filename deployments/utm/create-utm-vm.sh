#!/usr/bin/env bash

set -e
# This script creates a Windows 11 ARM64 UTM VM on macOS using the specified configuration.

# Manual argument parsing for portability
arch="arm64"
os="windows11"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--arch)
		if [[ -n "$2" && ("$2" == "amd64" || "$2" == "arm64") ]]; then
			arch="$2"
			shift 2
		else
			echo "Invalid value for --arch: $2. Allowed values are 'amd64' or 'arm64'." >&2
			exit 1
		fi
		;;
	--os)
		if [[ -n "$2" && ("$2" == "windows11") ]]; then
			os="$2"
			shift 2
		elif [[ -n "$2" && ("$2" == "ubuntu-desktop" || "$2" == "ubuntu-server") ]]; then
			os="$2"
			shift 2
		else
			echo "Invalid value for --os: $2. Allowed values are 'windows11', 'ubuntu-desktop', or 'ubuntu-server'." >&2
			exit 1
		fi
		;;
	--verbose)
		set -x
		shift
		;;
	*)
		echo "Unknown option: $1" >&2
		exit 1
		;;
	esac
done

script_dir=$(
	cd "$(dirname "$0")"
	pwd
)
project_root=$(
	cd "${script_dir}/../.."
	pwd
)

cache_dir="$project_root/cache"

utm_vm_dir="/Users/$(whoami)/Library/Containers/com.utmapp.UTM/Data/Documents"

qemu_img="$cache_dir/windows/qemu-windows11-$arch.qcow2"
if [[ "$os" == ubuntu* ]]; then
	qemu_img="$cache_dir/linux/linux-$os-qemu-$arch/linux-$os-packer.qcow2"
fi

if [ -d "$utm_vm_dir" ] && [ ! -d "$utm_vm_dir/$os-$arch-dev-alchemy.utm/Data" ]; then
	mkdir -p "$utm_vm_dir/$os-$arch-dev-alchemy.utm/Data"
fi

if [ "$os" = "Windows11" ] && [ ! -f "$qemu_img" ]; then
	echo "The required qcow2 image for Windows 11 $arch is missing. Please run the create-qemu-qcow2-disk.sh script first."
	exit 1
elif [[ "$os" == ubuntu* ]] && [ ! -f "$qemu_img" ]; then
	echo "The required qcow2 image for Ubuntu $arch is missing. Please run the packer build for with linux-ubuntu-on-macos.sh first."
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

os_lower=$(echo "$os" | awk '{print tolower()}')
os_lower_first_part=$(echo "$os_lower" | cut -d'-' -f1)

if [ ! -f "$utm_vm_dir/$os-$arch-dev-alchemy.utm/config.plist" ]; then
	echo "Creating UTM VM config..."

	QCOW_IMAGE="$os-$arch.qcow2" \
		VM_NAME="$os-$arch-dev-alchemy" \
		MAC_ADDRESS="$mac_address" \
		UUID="$(generate_uuid)" \
		UUID_CD="$(generate_uuid)" \
		UUID_DISK="$(generate_uuid)" \
		envsubst <"$project_root/deployments/utm/$os_lower_first_part-$arch/config.plist" >"$utm_vm_dir/$os-$arch-dev-alchemy.utm/config.plist"
fi

if [ ! -f "$utm_vm_dir/$os-$arch-dev-alchemy.utm/Data/$os_lower-$arch.qcow2" ]; then
	echo "Copying the qcow2 image to UTM VM directory..."
	rsync -av --progress "$qemu_img" "$utm_vm_dir/$os-$arch-dev-alchemy.utm/Data/$os_lower-$arch.qcow2"
else
	echo "The qcow2 image already exists in the UTM VM directory. Skipping copy."
fi
echo "$os $arch UTM VM setup is complete. You can now open UTM and start the VM."
