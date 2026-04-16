#!/bin/bash

set -e

SCRIPT_DIR=$(
	cd "$(dirname "$0")"
	pwd
)

host_os="$(uname -s)"
if [ "$host_os" = "Darwin" ]; then
	default_app_data_dir="$HOME/Library/Application Support/dev-alchemy"
else
	default_app_data_dir="${XDG_DATA_HOME:-$HOME/.local/share}/dev-alchemy"
fi

app_data_dir="${DEV_ALCHEMY_APP_DATA_DIR:-$default_app_data_dir}"
cache_dir="${DEV_ALCHEMY_CACHE_DIR:-$app_data_dir/cache}"
export DEV_ALCHEMY_APP_DATA_DIR="$app_data_dir"
export DEV_ALCHEMY_CACHE_DIR="$cache_dir"

DEB_PATH="${cache_dir}/qemu-efi-aarch64_all.deb"

# Ensure cache directory exists before downloading
mkdir -p "${cache_dir}"

if [ ! -f "${DEB_PATH}" ]; then
	echo "Resolving latest qemu-efi-aarch64 download URL from Debian trixie package index"
	PACKAGES_GZ_URL="https://deb.debian.org/debian/dists/trixie/main/binary-all/Packages.gz"
	FILENAME=$(curl -sL "${PACKAGES_GZ_URL}" | gunzip | awk '/^Package: qemu-efi-aarch64$/{found=1} found && /^Filename:/{print $2; exit}')
	if [ -z "${FILENAME}" ]; then
		echo "Failed to resolve qemu-efi-aarch64 package URL from Debian index" >&2
		exit 1
	fi
	DOWNLOAD_URL="https://deb.debian.org/debian/${FILENAME}"
	echo "Downloading ${DOWNLOAD_URL}"
	curl -o "${DEB_PATH}" -L "${DOWNLOAD_URL}"
else
	echo "qemu-efi-aarch64_all.deb already exists, skipping download"
fi

mkdir -p "${cache_dir}/qemu-uefi"
if [ ! -f "${cache_dir}/qemu-uefi/data.tar.xz" ]; then
	echo "Extract qemu-uefi data.tar.xz"
	tar -xvf "${DEB_PATH}" -C "${cache_dir}/qemu-uefi"
else
	echo "qemu-uefi/data.tar.xz already exists, skipping extraction"
fi

if [ ! -d "${cache_dir}/qemu-uefi/usr/share/qemu-efi-aarch64" ]; then
	tar -xvf "${cache_dir}/qemu-uefi/data.tar.xz" -C "${cache_dir}/qemu-uefi"
else
	echo "qemu-uefi/usr/share/qemu-efi-aarch64 already exists, skipping extraction"
fi
