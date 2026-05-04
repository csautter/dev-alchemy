#!/usr/bin/env bash

set -e

SCRIPT_DIR=$(
	cd "$(dirname "$0")" || exit 1
	pwd -P
)

host_os="$(uname -s)"
if [[ "$host_os" == "Darwin" ]]; then
	default_app_data_dir="$HOME/Library/Application Support/dev-alchemy"
elif [[ "$host_os" == "Linux" ]]; then
	default_app_data_dir="${XDG_DATA_HOME:-$HOME/.local/share}/dev-alchemy"
else
	echo "Unsupported host OS for Windows ARM64 unattended ISO creation: $host_os" >&2
	exit 1
fi

app_data_dir="${DEV_ALCHEMY_APP_DATA_DIR:-$default_app_data_dir}"
cache_dir="${DEV_ALCHEMY_CACHE_DIR:-$app_data_dir/cache}"
export DEV_ALCHEMY_APP_DATA_DIR="$app_data_dir"
export DEV_ALCHEMY_CACHE_DIR="$cache_dir"

vendor_dir="$cache_dir/windows"
iso_dir="$cache_dir/windows11/iso"
autounattend_xml_path="$SCRIPT_DIR/../../build/packer/windows/qemu-arm64/autounattend.xml"
windows_source_iso_path="$iso_dir/win11_25h2_english_arm64.iso"
windows_target_iso_path="$iso_dir/Win11_ARM64_Unattended.iso"
windows_extract_dir="$vendor_dir/win11_arm64_iso_files"

if [[ ! -f "$windows_source_iso_path" ]]; then
	echo "Windows 11 ARM64 source ISO is missing: $windows_source_iso_path" >&2
	exit 1
fi
if ! command -v xorriso >/dev/null 2>&1; then
	echo "xorriso is required to create the Windows 11 ARM64 unattended ISO." >&2
	exit 1
fi

mkdir -p "$vendor_dir" "$iso_dir"
rm -f "$vendor_dir/autounattend.iso"
xorriso -as mkisofs \
	-V autounattend \
	-o "$vendor_dir/autounattend.iso" \
	"$SCRIPT_DIR/../../build/packer/windows/qemu-arm64/"

rm -rf "$windows_extract_dir"
mkdir -p "$windows_extract_dir"

echo "Extracting Windows 11 ARM64 source ISO..."
xorriso -osirrox on \
	-indev "$windows_source_iso_path" \
	-extract / "$windows_extract_dir"
chmod -R u+w "$windows_extract_dir"

cp "$autounattend_xml_path" "$windows_extract_dir/autounattend.xml"
ls "$windows_extract_dir/autounattend.xml"

rm -f "$windows_target_iso_path"

source_size_kb=$(du -sk "$windows_extract_dir" | awk '{print $1}')
required_space_kb=$((source_size_kb * 15 / 10))
available_space_kb=$(df -k "$iso_dir" | awk 'NR==2 {print $4}')
echo "Disk space check:"
echo "  Source files: $((source_size_kb / 1024)) MB"
echo "  Required: $((required_space_kb / 1024)) MB"
echo "  Available: $((available_space_kb / 1024)) MB"
if [[ "$available_space_kb" -lt "$required_space_kb" ]]; then
	echo "ERROR: Not enough disk space to create the ISO image." >&2
	exit 1
fi

xorriso -as mkisofs \
	-iso-level 3 \
	-volid "CCCOMA_A64FRE_EN_US_DV9" \
	-eltorito-alt-boot \
	-e --interval:appended_partition_2:all:: \
	-no-emul-boot \
	-isohybrid-gpt-basdat \
	-append_partition 2 0xef "$windows_extract_dir/efi/microsoft/boot/efisys.bin" \
	-o "$windows_target_iso_path" \
	"$windows_extract_dir"

echo "Created $windows_target_iso_path"
rm -rf "$windows_extract_dir"
