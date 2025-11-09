#!/usr/bin/env bash

set +e # we want to continue on errors

# Manual argument parsing for portability
arch="arm64"
headless="false"
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
	--vnc-port)
		if [[ -n "$2" && "$2" =~ ^[0-9]+$ ]]; then
			vnc_port="$2"
			shift 2
		else
			echo "Invalid value for --vnc-port: $2. It must be a number." >&2
			exit 1
		fi
		;;
	--headless)
		headless="true"
		shift
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

echo "Using architecture: $arch"
echo "Headless mode: $headless"

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

cd "${project_root}" || exit 1

# download the Windows 11 ISO if not already present
if [ ! -d ./vendor/windows ]; then
	mkdir -p ./vendor/windows
fi
if [ ! -f "./vendor/windows/win11_25H2_english_$arch.iso" ]; then
	echo "Downloading Windows 11 $arch ISO"
	cd "${project_root}/scripts/macos/" || exit 1
	if [ ! -d .venv ]; then
		python3 -m venv .venv
	fi
	# shellcheck disable=SC1091
	source .venv/bin/activate

	if ! python -c "import playwright" &>/dev/null; then
		pip install playwright
		python -m playwright install chromium
	fi

	if [ "$arch" = "amd64" ]; then
		python playwright_win11_iso.py
	elif [ "$arch" = "arm64" ]; then
		python playwright_win11_iso.py --arm
	fi
	cd "${project_root}/vendor/windows/" || exit 1

	if [ "$headless" = "true" ]; then
		echo "Running in headless mode, skipping ISO download progress bar"
		curl -o "win11_25h2_english_$arch.iso" "$(cat "./win11_${arch}_iso_url.txt")"
	else
		echo "Running in interactive mode, showing ISO download progress bar"
		curl --progress-bar -o "win11_25h2_english_$arch.iso" "$(cat "./win11_${arch}_iso_url.txt")"
	fi

	cd "${project_root}" || exit 1
else
	echo "Windows 11 $arch ISO already exists, skipping download"
fi

bash scripts/macos/download-utm-guest-tools.sh

if [ "$arch" = "arm64" ]; then
	# download the qemu-uefi files if not already present
	bash scripts/macos/download-arm64-uefi.sh

	# builds the autounattend ISO with the current autounattend.xml file
	bash scripts/macos/create-win11-autounattend-iso.sh

	# download the virtio-win ISO if not already present
	bash scripts/macos/download-virtio-win-iso.sh

fi

# creates the qcow2 disk image and overwrites it if it already exists
bash scripts/macos/create-qemu-qcow2-disk.sh --arch $arch

packer init "build/packer/windows/windows11-on-macos.pkr.hcl"

# determine the Windows 11 ISO path to use
if [ "$arch" = "amd64" ]; then
	win11_iso_path="${project_root}/vendor/windows/win11_25h2_english_$arch.iso"
elif [ "$arch" = "arm64" ]; then
	# use the unattended ISO we created earlier
	win11_iso_path="${project_root}/vendor/windows/Win11_ARM64_Unattended.iso"
fi

# remove packer output directory if it exists
output_dir="$project_root/cache/windows11/qemu-out-windows11-${arch}"
if [ -d "$output_dir" ]; then
	echo "Removing existing Packer output directory..."
	rm -rf "$output_dir"
fi

if [ "$verbose" = "true" ]; then
	export PACKER_LOG=1
fi
packer build -var "iso_url=${win11_iso_path}" -var "headless=$headless" -var "vnc_port=$vnc_port" -var "arch=$arch" "build/packer/windows/windows11-on-macos.pkr.hcl"
packer_exit_code=$?

exit $packer_exit_code
