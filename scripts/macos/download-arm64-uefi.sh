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
UEFI_DIR="${cache_dir}/qemu-uefi"
UEFI_FIRMWARE_PATH="${UEFI_DIR}/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"

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
	curl -fL -o "${DEB_PATH}" "${DOWNLOAD_URL}"
else
	echo "qemu-efi-aarch64_all.deb already exists, skipping download"
fi

if [ ! -f "${UEFI_FIRMWARE_PATH}" ]; then
	echo "Extracting qemu-efi-aarch64 package contents"
	tmp_extract_dir="$(mktemp -d "${cache_dir}/qemu-uefi.tmp.XXXXXX")"
	tmp_rootfs_dir="${tmp_extract_dir}/rootfs"
	trap 'rm -rf "${tmp_extract_dir}"' EXIT

	if command -v dpkg-deb >/dev/null 2>&1; then
		mkdir -p "${tmp_rootfs_dir}"
		dpkg-deb -x "${DEB_PATH}" "${tmp_rootfs_dir}"
	else
		tmp_archive_dir="${tmp_extract_dir}/deb"
		mkdir -p "${tmp_archive_dir}" "${tmp_rootfs_dir}"
		(
			cd "${tmp_archive_dir}"
			ar x "${DEB_PATH}"
		)

		data_archive="$(find "${tmp_archive_dir}" -maxdepth 1 -type f -name 'data.tar.*' | head -n 1)"
		if [ -z "${data_archive}" ]; then
			echo "Failed to locate data archive inside ${DEB_PATH}" >&2
			exit 1
		fi
		tar -xf "${data_archive}" -C "${tmp_rootfs_dir}"
	fi

	rm -rf "${UEFI_DIR}"
	mv "${tmp_rootfs_dir}" "${UEFI_DIR}"
	rm -rf "${tmp_extract_dir}"
	trap - EXIT
else
	echo "qemu-uefi firmware already exists, skipping extraction"
fi

if [ ! -f "${UEFI_FIRMWARE_PATH}" ]; then
	echo "Expected ARM64 UEFI firmware was not found at ${UEFI_FIRMWARE_PATH}" >&2
	exit 1
fi
