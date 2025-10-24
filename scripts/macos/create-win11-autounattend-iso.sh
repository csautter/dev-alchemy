#!/bin/bash

set -ex

SCRIPT_DIR=$(
	cd "$(dirname "$0")"
	pwd
)

vendor_dir="$SCRIPT_DIR/../../vendor/windows"
autounattend_xml_path="$SCRIPT_DIR/../../build/packer/windows/qemu-arm64/autounattend.xml"
windows_source_iso_path="$vendor_dir/win11_25h2_english_arm64.iso"
windows_target_iso_path="$vendor_dir/Win11_ARM64_Unattended.iso"

# Create the autounattend ISO
rm -f "$vendor_dir/autounattend.iso"
/usr/bin/hdiutil makehybrid -o "$vendor_dir/autounattend.iso" -hfs -joliet -iso -default-volume-name autounattend "$SCRIPT_DIR/../../build/packer/windows/qemu-arm64/"

# Create a Windows 11 ARM64 ISO with the autounattend.xml merged in
mkdir -p "$vendor_dir/win11_arm64_source_iso"
hdiutil mount -mountpoint "$vendor_dir/win11_arm64_source_iso" "$windows_source_iso_path"

mkdir -p "$vendor_dir/win11_arm64_iso_files"
echo "Enter sudo password if asked to copy ISO files..."
sudo cp -a "$vendor_dir/win11_arm64_source_iso/"* "$vendor_dir/win11_arm64_iso_files/"
hdiutil unmount "$vendor_dir/win11_arm64_source_iso"

cp "$autounattend_xml_path" "$vendor_dir/win11_arm64_iso_files/autounattend.xml"
ls "$vendor_dir/win11_arm64_iso_files/autounattend.xml"

rm -f "$windows_target_iso_path"

# Create the target ISO with the autounattend.xml file
xorriso -as mkisofs \
	-iso-level 3 \
	-volid "CCCOMA_A64FRE_EN-US_DV9" \
	-eltorito-alt-boot \
	-e --interval:appended_partition_2:all:: \
	-no-emul-boot \
	-isohybrid-gpt-basdat \
	-append_partition 2 0xef "$vendor_dir/win11_arm64_iso_files/efi/microsoft/boot/efisys.bin" \
	-o "$windows_target_iso_path" \
	"$vendor_dir/win11_arm64_iso_files"

echo "--------------------------------"
echo "Debug: Listing contents of source ISO:"
xorriso -indev "$windows_source_iso_path" -report_el_torito plain
echo "--------------------------------"

echo "Debug: Listing contents of created ISO:"
xorriso -indev "$windows_target_iso_path" -report_el_torito plain
echo "--------------------------------"

# clean up
#sudo rm -rf "$vendor_dir/win11_arm64_source_iso"
sudo rm -rf "$vendor_dir/win11_arm64_iso_files"
