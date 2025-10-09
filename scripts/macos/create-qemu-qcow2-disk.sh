#!/bin/bash

set -ex

# create a qcow2 disk image for QEMU
SCRIPT_DIR=$(cd $(dirname $0); pwd)
rm -f $SCRIPT_DIR/../../vendor/windows/qemu-windows11-arm64.qcow2
qemu-img create -f qcow2 -o compression_type=zstd $SCRIPT_DIR/../../vendor/windows/qemu-windows11-arm64.qcow2 64G
qemu-img info $SCRIPT_DIR/../../vendor/windows/qemu-windows11-arm64.qcow2