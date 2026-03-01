#!/bin/bash

set -e

SCRIPT_DIR=$(
	cd "$(dirname "$0")"
	pwd
)

DEB_PATH="${SCRIPT_DIR}/../../cache/qemu-efi-aarch64_all.deb"

# Ensure cache directory exists before downloading
mkdir -p "${SCRIPT_DIR}"/../../cache

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

mkdir -p "${SCRIPT_DIR}"/../../cache/qemu-uefi
if [ ! -f "${SCRIPT_DIR}"/../../cache/qemu-uefi/data.tar.xz ]; then
	echo "Extract qemu-uefi data.tar.xz"
	tar -xvf "${DEB_PATH}" -C "${SCRIPT_DIR}"/../../cache/qemu-uefi
else
	echo "qemu-uefi/data.tar.xz already exists, skipping extraction"
fi

if [ ! -d "${SCRIPT_DIR}"/../../cache/qemu-uefi/usr/share/qemu-efi-aarch64 ]; then
	tar -xvf "${SCRIPT_DIR}"/../../cache/qemu-uefi/data.tar.xz -C "${SCRIPT_DIR}"/../../cache/qemu-uefi
else
	echo "qemu-uefi/usr/share/qemu-efi-aarch64 already exists, skipping extraction"
fi
