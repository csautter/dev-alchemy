#!/usr/bin/env bash

set +e
set -x

arch="arm64"
headless="false"
ubuntu_type="server"
vnc_port="5901"
cpus="4"
memory="4096"
verbose="false"
build_output_dir=""

script_dir=$(
	cd "$(dirname "$0")"
	pwd -P
)
project_root=$(
	cd "${script_dir}/../../../.."
	pwd -P
)

file_size_bytes() {
	if [[ ! -f "$1" ]]; then
		echo "0"
		return 0
	fi
	stat -f%z "$1"
}

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
	--cpus)
		if [[ -n "$2" && "$2" =~ ^[0-9]+$ ]]; then
			cpus="$2"
			shift 2
		else
			echo "Invalid value for --cpus: $2. It must be a number." >&2
			exit 1
		fi
		;;
	--memory)
		if [[ -n "$2" && "$2" =~ ^[0-9]+$ ]]; then
			memory="$2"
			shift 2
		else
			echo "Invalid value for --memory: $2. It must be a number." >&2
			exit 1
		fi
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
	--project-root)
		if [[ -n "$2" ]]; then
			project_root="$2"
			shift 2
		else
			echo "Invalid value for --project-root: $2." >&2
			exit 1
		fi
		;;
	--build-output-dir)
		if [[ -n "$2" ]]; then
			build_output_dir="$2"
			shift 2
		else
			echo "Invalid value for --build-output-dir: $2." >&2
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

if [[ "$(uname -s)" != "Darwin" ]]; then
	echo "This script only supports macOS hosts. Use linux-ubuntu-on-linux.sh on Linux." >&2
	exit 1
fi

if [[ -z "$build_output_dir" ]]; then
	build_output_dir="/tmp/dev-alchemy/qemu-out-ubuntu-${ubuntu_type}-${arch}"
fi

app_data_dir="${DEV_ALCHEMY_APP_DATA_DIR:-$HOME/Library/Application Support/dev-alchemy}"
cache_dir="${DEV_ALCHEMY_CACHE_DIR:-$app_data_dir/cache}"
packer_cache_dir="${DEV_ALCHEMY_PACKER_CACHE_DIR:-$app_data_dir/packer_cache}"

mkdir -p "$cache_dir" "$packer_cache_dir"
export DEV_ALCHEMY_APP_DATA_DIR="$app_data_dir"
export DEV_ALCHEMY_CACHE_DIR="$cache_dir"
export DEV_ALCHEMY_PACKER_CACHE_DIR="$packer_cache_dir"
export PACKER_CACHE_DIR="$packer_cache_dir"

if [[ "$arch" == "arm64" ]]; then
	if ! bash "$project_root/scripts/macos/download-arm64-uefi.sh"; then
		echo "Failed to prepare ARM64 UEFI firmware." >&2
		exit 1
	fi
	firmware_path="$cache_dir/qemu-uefi/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"
	if [[ ! -f "$firmware_path" ]]; then
		echo "ARM64 UEFI firmware is missing: $firmware_path" >&2
		exit 1
	fi
fi

iso_path="$cache_dir/linux/ubuntu-24.04.3-live-server-amd64.iso"
if [[ "$arch" == "arm64" ]]; then
	iso_path="$cache_dir/linux/ubuntu-24.04.3-live-server-arm64.iso"
	iso_url="https://cdimage.ubuntu.com/releases/24.04.3/release/ubuntu-24.04.3-live-server-arm64.iso"
	iso_checksum="2ee2163c9b901ff5926400e80759088ff3b879982a3956c02100495b489fd555"
	mkdir -p "$(dirname "$iso_path")"

	if [[ ! -f "$iso_path" || "$(file_size_bytes "$iso_path")" -lt 2500000000 ]]; then
		echo "Downloading Ubuntu ISO (supports resume)..."
		if ! curl --no-buffer --retry 10 --continue-at - -L -# -o "$iso_path" "$iso_url"; then
			echo "Failed to download Ubuntu ISO." >&2
			exit 1
		fi
	fi

	echo "Verifying ISO checksum..."
	downloaded_checksum=$(shasum -a 256 "$iso_path" | awk '{print $1}')
	if [[ "$downloaded_checksum" != "$iso_checksum" ]]; then
		echo "Checksum mismatch for $iso_path" >&2
		exit 1
	fi
fi

if [[ "$arch" == "arm64" ]]; then
	echo "Creating QCOW2 disk image..."
	output_directory="$cache_dir/ubuntu"
	mkdir -p "$output_directory"
	echo "Removing existing QCOW2 disk image if it exists..."
	rm -f "$output_directory/qemu-ubuntu-${ubuntu_type}-packer-${arch}.qcow2"
	qemu-img create -f qcow2 -o compression_type=zstd "$output_directory/qemu-ubuntu-${ubuntu_type}-packer-${arch}.qcow2" 64G
	qemu-img info "$output_directory/qemu-ubuntu-${ubuntu_type}-packer-${arch}.qcow2"
fi

if [[ "$arch" == "arm64" ]]; then
	cd "$project_root/build/packer/linux/ubuntu/cloud-init/qemu-${ubuntu_type}" || exit 1
	rm -f cidata.iso
	xorriso -as mkisofs -V cidata -o cidata.iso user-data meta-data
	cd "$project_root" || exit 1
fi

output_dir="$build_output_dir"
if [[ -d "$output_dir" ]]; then
	echo "Removing existing Packer output directory..."
	rm -rf "$output_dir"
fi
mkdir -p "$(dirname "$output_dir")"

packer_file="build/packer/linux/ubuntu/linux-ubuntu-on-macos.pkr.hcl"
packer init "$packer_file"

if [[ "$verbose" == "true" ]]; then
	export PACKER_LOG=1
fi

packer build \
	-var "cache_dir=$cache_dir" \
	-var "build_output_dir=$build_output_dir" \
	-var "iso_url=$iso_path" \
	-var "ubuntu_type=$ubuntu_type" \
	-var "headless=$headless" \
	-var "vnc_port=$vnc_port" \
	-var "arch=$arch" \
	-var "cpus=$cpus" \
	-var "memory=$memory" \
	"$packer_file"
