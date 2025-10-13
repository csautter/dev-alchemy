#!/bin/bash

set -ex

SCRIPT_DIR=$(
	cd $(dirname $0)
	pwd
)
if [ ! -f ${SCRIPT_DIR}/../../vendor/qemu-efi-aarch64_2025.02-10_all.deb ]; then
	echo "Downloading qemu-efi-aarch64_2025.02-10_all.deb"
	curl -o ${SCRIPT_DIR}/../../vendor/qemu-efi-aarch64_2025.02-10_all.deb -L http://ftp.de.debian.org/debian/pool/main/e/edk2/qemu-efi-aarch64_2025.02-10_all.deb
else
	echo "qemu-efi-aarch64_2025.02-10_all.deb already exists, skipping download"
fi

mkdir -p ${SCRIPT_DIR}/../../vendor/qemu-uefi
if [ ! -f ${SCRIPT_DIR}/../../vendor/qemu-uefi/data.tar.xz ]; then
	echo "Extract qemu-uefi data.tar.xz"
	tar -xvf ${SCRIPT_DIR}/../../vendor/qemu-efi-aarch64_2025.02-10_all.deb -C ${SCRIPT_DIR}/../../vendor/qemu-uefi
else
	echo "qemu-uefi/data.tar.xz already exists, skipping extraction"
fi

if [ ! -d ${SCRIPT_DIR}/../../vendor/qemu-uefi/usr/share/qemu-efi-aarch64 ]; then
	tar -xvf ${SCRIPT_DIR}/../../vendor/qemu-uefi/data.tar.xz -C ${SCRIPT_DIR}/../../vendor/qemu-uefi
else
	echo "qemu-uefi/usr/share/qemu-efi-aarch64 already exists, skipping extraction"
fi
