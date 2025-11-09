#!/usr/bin/env bash

set +e # we want to continue on errors
set -x

# Manual argument parsing for portability
arch="arm64"
headless="false"
ubuntu_type="server"
vnc_port="5901"
verbose="false"

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
	--headless)
		headless="true"
		shift
		;;
	--vnc-port)
		if [[ -n "$2" && "$2" =~ ^[0-9]+$ ]]; then
			vnc_port="$2"
			shift 2
		else
			echo "Invalid value for --vnc-port: $2. It must be a number." >&2
			exit 1
		fi
		;;
	--ubuntu-type)
		if [[ -n "$2" && ("$2" == "server" || "$2" == "desktop") ]]; then
			ubuntu_type="$2"
			shift 2
		else
			echo "Invalid value for --ubuntu-type: $2. Allowed values are 'server' or 'desktop'." >&2
			exit 1
		fi
		;;
	--verbose)
		set -x
		verbose="true"
		shift
		;;
	*)
		echo "Unknown option: $1" >&2
		exit 1
		;;
	esac
done

script_dir=$(
	# shellcheck disable=SC2164
	cd "$(dirname "$0")"
	pwd -P
)
project_root=$(
	# shellcheck disable=SC2164
	cd "${script_dir}/../../../.."
	pwd -P
)
cache_dir="$project_root/cache"

# Download uefi-firmware if it doesn't exist
if [ "$arch" = "arm64" ]; then
	bash "$project_root/scripts/macos/download-arm64-uefi.sh"
fi

# Download the Ubuntu ISO if it doesn't exist
if [ "$arch" = "arm64" ]; then
	iso_path="$project_root/vendor/linux/ubuntu-24.04.3-live-server-arm64.iso"
	iso_url="https://cdimage.ubuntu.com/releases/24.04.3/release/ubuntu-24.04.3-live-server-arm64.iso"
	iso_checksum="2ee2163c9b901ff5926400e80759088ff3b879982a3956c02100495b489fd555"
	mkdir -p "$(dirname "$iso_path")"

	if [[ ! -f "$iso_path" ]]; then
		echo "Downloading Ubuntu ISO..."
		curl -L -o "$iso_path" "$iso_url"
	fi

	echo "Verifying ISO checksum..."
	downloaded_checksum=$(sha256sum "$iso_path" | awk '{print $1}')
	if [[ "$downloaded_checksum" != "$iso_checksum" ]]; then
		echo "Checksum mismatch for $iso_path" >&2
		exit 1
	fi
fi

# creates the qcow2 disk image and overwrites it if it already exists
if [ "$arch" = "arm64" ]; then
	echo "Creating QCOW2 disk image..."
	output_directory="$cache_dir/ubuntu/linux-ubuntu-${ubuntu_type}-qemu-${arch}"
	mkdir -p "$output_directory"
	echo "Removing existing QCOW2 disk image if it exists..."
	rm -f "$output_directory/linux-ubuntu-${ubuntu_type}-packer.qcow2"
	qemu-img create -f qcow2 -o compression_type=zstd "$output_directory/linux-ubuntu-${ubuntu_type}-packer.qcow2" 64G
	qemu-img info "$output_directory/linux-ubuntu-${ubuntu_type}-packer.qcow2"
fi

# create cidata iso
if [ "$arch" = "arm64" ]; then
	cd "$script_dir/cloud-init/qemu-${ubuntu_type}" || exit 1
	rm -f cidata.iso
	xorriso -as mkisofs -V cidata -o cidata.iso user-data meta-data
	cd "$project_root" || exit 1
fi

# remove packer output directory if it exists
output_dir="$cache_dir/ubuntu/qemu-ubuntu-${ubuntu_type}-out-${arch}"
if [ -d "$output_dir" ]; then
	echo "Removing existing Packer output directory..."
	rm -rf "$output_dir"
fi

packer init "build/packer/linux/ubuntu/linux-ubuntu-on-macos.pkr.hcl"

if [ "$verbose" = "true" ]; then
	export PACKER_LOG=1
fi
packer build -var "ubuntu_type=$ubuntu_type" -var "headless=$headless" -var "vnc_port=$vnc_port" -var "arch=$arch" "build/packer/linux/ubuntu/linux-ubuntu-on-macos.pkr.hcl"
