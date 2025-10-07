#!/bin/bash

#/usr/bin/hdiutil [makehybrid -o /var/folders/vn/qsn35_f50c3bc6h7ffm3r_cm0000gn/T/packer1177975043.iso -hfs -joliet -iso -default-volume-name packer /var/folders/vn/qsn35_f50c3bc6h7ffm3r_cm0000gn/T/packer_to_cdrom1393625311]
SCRIPT_DIR=$(cd $(dirname $0); pwd)
rm -f $SCRIPT_DIR/../../vendor/windows/autounattend.iso
/usr/bin/hdiutil makehybrid -o $SCRIPT_DIR/../../vendor/windows/autounattend.iso -hfs -joliet -iso -default-volume-name autounattend $SCRIPT_DIR/../../build/packer/windows/qemu-arm64/