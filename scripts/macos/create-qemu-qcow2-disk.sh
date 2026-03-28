#!/bin/bash

set -ex

arch="arm64"

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
	*)
		echo "Unknown option: $1" >&2
		exit 1
		;;
	esac
done

# create a qcow2 disk image for QEMU
script_dir=$(
	cd "$(dirname "$0")"
	pwd
)

app_data_dir="${DEV_ALCHEMY_APP_DATA_DIR:-$HOME/Library/Application Support/dev-alchemy}"
cache_dir="${DEV_ALCHEMY_CACHE_DIR:-$app_data_dir/cache}"
export DEV_ALCHEMY_APP_DATA_DIR="$app_data_dir"
export DEV_ALCHEMY_CACHE_DIR="$cache_dir"

mkdir -p "$cache_dir/windows11/"
rm -f "$cache_dir/windows11/qemu-windows11-$arch.qcow2"
qemu-img create -f qcow2 -o compression_type=zstd "$cache_dir/windows11/qemu-windows11-$arch.qcow2" 64G
qemu-img info "$cache_dir/windows11/qemu-windows11-$arch.qcow2"
