#!/usr/bin/env bash

set +e # we want to continue on errors

# Manual argument parsing for portability
arch="arm64"
headless="false"
vnc_port="5901"
cpus="4"
memory="4096"
verbose="false"
build_output_dir=""

script_dir=$(
	# shellcheck disable=SC2164
	cd "$(dirname "$0")"
	pwd -P
)
project_root=$(
	# shellcheck disable=SC2164
	cd "${script_dir}/../../.."
	pwd -P
)
app_data_dir="${DEV_ALCHEMY_APP_DATA_DIR:-$HOME/Library/Application Support/dev-alchemy}"
cache_dir="${DEV_ALCHEMY_CACHE_DIR:-$app_data_dir/cache}"
packer_cache_dir="${DEV_ALCHEMY_PACKER_CACHE_DIR:-$app_data_dir/packer_cache}"

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
	--vnc-port)
		if [[ -n "$2" && "$2" =~ ^[0-9]+$ ]]; then
			vnc_port="$2"
			shift 2
		else
			echo "Invalid value for --vnc-port: $2. It must be a number." >&2
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
	--verbose)
		set -x
		verbose="true"
		shift
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
	*)
		echo "Unknown option: $1" >&2
		exit 1
		;;
	esac
done

if [[ -z "$build_output_dir" ]]; then
	build_output_dir="/tmp/dev-alchemy/qemu-out-windows11-${arch}"
fi

echo "Using architecture: $arch"
echo "Headless mode: $headless"

cd "${project_root}" || exit 1
mkdir -p "$cache_dir" "$packer_cache_dir"
export DEV_ALCHEMY_APP_DATA_DIR="$app_data_dir"
export DEV_ALCHEMY_CACHE_DIR="$cache_dir"
export DEV_ALCHEMY_PACKER_CACHE_DIR="$packer_cache_dir"
export PACKER_CACHE_DIR="$packer_cache_dir"

# download the Windows 11 ISO if not already present
if [ ! -d "$cache_dir/windows11/iso" ]; then
	mkdir -p "$cache_dir/windows11/iso"
fi
if [ ! -f "$cache_dir/windows11/iso/win11_25h2_english_$arch.iso" ]; then
	echo "Downloading Windows 11 $arch ISO"
	cd "${project_root}/scripts/macos/" || exit 1
	if [ ! -d .venv ]; then
		python3 -m venv .venv
	fi
	# shellcheck disable=SC1091
	source .venv/bin/activate

	pip install -r requirements.txt
	python -m playwright install chromium

	if [ "$arch" = "amd64" ]; then
		python playwright_win11_iso.py
	elif [ "$arch" = "arm64" ]; then
		python playwright_win11_iso.py --arm
	fi
	mkdir -p "${cache_dir}/windows11/iso"
	cd "${cache_dir}/windows/" || exit 1

	if [ "$headless" = "true" ]; then
		echo "Running in headless mode, skipping ISO download progress bar"
		curl -o "${cache_dir}/windows11/iso/win11_25h2_english_$arch.iso" "$(cat "./win11_${arch}_iso_url.txt")"
	else
		echo "Running in interactive mode, showing ISO download progress bar"
		curl --progress-bar -o "${cache_dir}/windows11/iso/win11_25h2_english_$arch.iso" "$(cat "./win11_${arch}_iso_url.txt")"
	fi

	cd "${project_root}" || exit 1
else
	echo "Windows 11 $arch ISO already exists, skipping download"
fi

bash "${project_root}/scripts/macos/download-utm-guest-tools.sh"

if [ "$arch" = "arm64" ]; then
	# download the qemu-uefi files if not already present
	if ! bash "${project_root}/scripts/macos/download-arm64-uefi.sh"; then
		echo "Failed to prepare ARM64 UEFI firmware." >&2
		exit 1
	fi
	firmware_path="${cache_dir}/qemu-uefi/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"
	if [ ! -f "${firmware_path}" ]; then
		echo "ARM64 UEFI firmware is missing: ${firmware_path}" >&2
		exit 1
	fi

	# builds the autounattend ISO with the current autounattend.xml file
	echo "Creating Windows 11 ARM64 unattended ISO..."
	echo "Running the create-win11-autounattend-iso.sh script to generate the unattended ISO..."
	bash "${project_root}/scripts/macos/create-win11-autounattend-iso.sh"

	# download the virtio-win ISO if not already present
	bash "${project_root}/scripts/macos/download-virtio-win-iso.sh"

fi

# creates the qcow2 disk image and overwrites it if it already exists
bash "${project_root}/scripts/macos/create-qemu-qcow2-disk.sh" --arch "$arch"

packer init "${project_root}/build/packer/windows/windows11-on-macos.pkr.hcl"

# determine the Windows 11 ISO path to use
if [ "$arch" = "amd64" ]; then
	win11_iso_path="${cache_dir}/windows11/iso/win11_25h2_english_$arch.iso"
elif [ "$arch" = "arm64" ]; then
	# use the unattended ISO we created earlier
	win11_iso_path="${cache_dir}/windows11/iso/Win11_ARM64_Unattended.iso"
fi

# remove packer output directory if it exists
output_dir="${build_output_dir}"
if [ -d "$output_dir" ]; then
	echo "Removing existing Packer output directory..."
	rm -rf "$output_dir"
fi
mkdir -p "$(dirname "$output_dir")"

if [ "$verbose" = "true" ]; then
	export PACKER_LOG=1
fi
packer build -var "cache_dir=${cache_dir}" -var "build_output_dir=${build_output_dir}" -var "iso_url=${win11_iso_path}" -var "headless=$headless" -var "vnc_port=$vnc_port" -var "arch=$arch" -var "cpus=$cpus" -var "memory=$memory" "build/packer/windows/windows11-on-macos.pkr.hcl"
packer_exit_code=$?

exit $packer_exit_code
