#!/bin/bash

set -ex

SCRIPT_DIR=$(
	cd "$(dirname "$0")"
	pwd
)

app_data_dir="${DEV_ALCHEMY_APP_DATA_DIR:-$HOME/Library/Application Support/dev-alchemy}"
cache_dir="${DEV_ALCHEMY_CACHE_DIR:-$app_data_dir/cache}"
export DEV_ALCHEMY_APP_DATA_DIR="$app_data_dir"
export DEV_ALCHEMY_CACHE_DIR="$cache_dir"

vendor_dir="$cache_dir/windows"
iso_dir="$cache_dir/windows11/iso"
autounattend_xml_path="$SCRIPT_DIR/../../build/packer/windows/qemu-arm64/autounattend.xml"
windows_source_iso_path="$iso_dir/win11_25h2_english_arm64.iso"
windows_target_iso_path="$iso_dir/Win11_ARM64_Unattended.iso"

# Create the autounattend ISO
rm -f "$vendor_dir/autounattend.iso"
/usr/bin/hdiutil makehybrid -o "$vendor_dir/autounattend.iso" -hfs -joliet -iso -default-volume-name autounattend "$SCRIPT_DIR/../../build/packer/windows/qemu-arm64/"

# Create a Windows 11 ARM64 ISO with the autounattend.xml merged in
mkdir -p "$vendor_dir/win11_arm64_source_iso"
hdiutil mount -mountpoint "$vendor_dir/win11_arm64_source_iso" "$windows_source_iso_path"

mkdir -p "$vendor_dir/win11_arm64_iso_files"
echo "__DEV_ALCHEMY_SILENT_HELPERS_ON__"
echo "Enter sudo password if asked to copy ISO files..."
sudo cp -a "$vendor_dir/win11_arm64_source_iso/"* "$vendor_dir/win11_arm64_iso_files/"
echo "__DEV_ALCHEMY_SILENT_HELPERS_OFF__"
hdiutil unmount "$vendor_dir/win11_arm64_source_iso"

cp "$autounattend_xml_path" "$vendor_dir/win11_arm64_iso_files/autounattend.xml"
ls "$vendor_dir/win11_arm64_iso_files/autounattend.xml"

rm -f "$windows_target_iso_path"

# Check available disk space before creating the ISO
# The output ISO is roughly the same size as the source files;
# use a 50% buffer to cover ISO overhead and the efisys partition.
source_size_kb=$(du -sk "$vendor_dir/win11_arm64_iso_files" | awk '{print $1}')
required_space_kb=$((source_size_kb * 15 / 10))
iso_fs=$(df -k "$iso_dir" | awk 'NR==2 {print $1}')
available_space_kb=$(df -k "$iso_dir" | awk 'NR==2 {print $4}')
echo "Disk space check:"
echo "  Source files            : $((source_size_kb / 1024)) MB"
echo "  Required (incl. 50% overhead) : $((required_space_kb / 1024)) MB"
echo "  Available on $iso_fs : $((available_space_kb / 1024)) MB"
echo "  Full disk usage:"
df -h
if [ "$available_space_kb" -lt "$required_space_kb" ]; then
	echo "ERROR: Not enough disk space to create the ISO image."
	exit 1
fi

# Create the target ISO with the autounattend.xml file
xorriso -as mkisofs \
	-iso-level 3 \
	-volid "CCCOMA_A64FRE_EN_US_DV9" \
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
echo "__DEV_ALCHEMY_SILENT_HELPERS_ON__"
sudo rm -rf "$vendor_dir/win11_arm64_iso_files"
echo "__DEV_ALCHEMY_SILENT_HELPERS_OFF__"
