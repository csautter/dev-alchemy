#!/bin/bash

# This script creates a bootable macOS installer ISO.
# It requires the macOS installer app to be downloaded from the App Store.
# Usage: ./create-macos-iso.sh
# It can be usefull to create a macOS VM with QEMU/KVM or other virtualization software.

set -ex

# === CONFIG ===
MACOS_VERSION="Sequoia"               # Adjust for Monterey, Big Sur, etc.
MACOS_INSTALLER_VERSION_NUMBER="15.7" # Adjust accordingly
INSTALLER="/Applications/Install macOS $MACOS_VERSION.app"
DISK_NAME="macos_installer_$MACOS_VERSION"
VOLUME_NAME="macOS $MACOS_VERSION"
ISO_NAME="macos_installer_$MACOS_VERSION.iso"
BASE_DIR="./vendor/macos/"

# === CHECK ===
if [ ! -d "$INSTALLER" ]; then
	echo "❌ Installer not found at: $INSTALLER"
	echo "Download it from the Mac App Store first."
	# check last version with:
	# softwareupdate --list-full-installers
	softwareupdate --fetch-full-installer --full-installer-version "$MACOS_INSTALLER_VERSION_NUMBER"
fi

echo "🔧 Creating disk image..."
if [ -f "$BASE_DIR$DISK_NAME.dmg" ]; then
	echo "⚠️ Disk image already exists. Removing it..."
	rm "$BASE_DIR$DISK_NAME.dmg"
fi

hdiutil create -o "$BASE_DIR$DISK_NAME" -size 17200m -volname "$VOLUME_NAME" -layout SPUD -fs HFS+J || exit 1

echo "🔌 Mounting disk image..."
hdiutil attach "$BASE_DIR$DISK_NAME.dmg" -mountpoint /Volumes/"$VOLUME_NAME" || exit 1

echo "📦 Creating bootable installer..."
echo "__DEV_ALCHEMY_SILENT_HELPERS_ON__"
echo "Please enter your sudo password if prompted."
sudo "$INSTALLER/Contents/Resources/createinstallmedia" --volume /Volumes/"$VOLUME_NAME" --nointeraction || exit 1
echo "__DEV_ALCHEMY_SILENT_HELPERS_OFF__"

echo "🔌 Detaching disk..."
hdiutil detach /Volumes/"Install macOS $MACOS_VERSION" || exit 1

echo "📀 Converting to ISO..."
hdiutil convert "$BASE_DIR$DISK_NAME.dmg" -format UDTO -o "$BASE_DIR$ISO_NAME" || exit 1

echo "✏️ Renaming ISO..."
mv "$BASE_DIR$ISO_NAME.cdr" "$BASE_DIR$ISO_NAME" || exit 1

echo "📥 Downloading OpenCore ISO..."
cd "$BASE_DIR" || exit 1
curl -LO https://github.com/kholia/OSX-KVM/raw/master/OpenCore/OpenCore-Catalina.iso || exit 1

echo "✅ Done! Bootable ISO at: $BASE_DIR$ISO_NAME"
