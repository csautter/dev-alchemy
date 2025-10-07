#!/bin/bash

SCRIPT_DIR=$(cd $(dirname $0); pwd)
curl -o ${SCRIPT_DIR}/../../vendor/qemu-efi-aarch64_2025.02-10_all.deb -L http://ftp.de.debian.org/debian/pool/main/e/edk2/qemu-efi-aarch64_2025.02-10_all.deb

mkdir -p ${SCRIPT_DIR}/../../vendor/qemu-uefi
tar -xvf ${SCRIPT_DIR}/../../vendor/qemu-efi-aarch64_2025.02-10_all.deb -C ${SCRIPT_DIR}/../../vendor/qemu-uefi
tar -xvf ${SCRIPT_DIR}/../../vendor/qemu-uefi/data.tar.xz -C ${SCRIPT_DIR}/../../vendor/qemu-uefi