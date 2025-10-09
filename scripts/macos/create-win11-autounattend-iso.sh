#!/bin/bash

set -ex

SCRIPT_DIR=$(cd $(dirname $0); pwd)
rm -f $SCRIPT_DIR/../../vendor/windows/autounattend.iso
/usr/bin/hdiutil makehybrid -o $SCRIPT_DIR/../../vendor/windows/autounattend.iso -hfs -joliet -iso -default-volume-name autounattend $SCRIPT_DIR/../../build/packer/windows/qemu-arm64/