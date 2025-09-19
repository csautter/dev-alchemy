#!/bin/bash

# === CONFIG ===
MACOS_VERSION="Sequoia"  # Adjust for Monterey, Big Sur, etc.
INSTALLER="/Applications/Install macOS $MACOS_VERSION.app"
DISK_NAME="macos_installer_$MACOS_VERSION"
VOLUME_NAME="macOS $MACOS_VERSION"
ISO_NAME="macos_installer_$MACOS_VERSION.iso"
BASE_DIR="./vendor/macos/"

# === CHECK ===
if [ ! -d "$INSTALLER" ]; then
  echo "❌ Installer not found at: $INSTALLER"
  echo "Download it from the Mac App Store first."
  exit 1
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
echo "Please enter your sudo password if prompted."
sudo "$INSTALLER/Contents/Resources/createinstallmedia" --volume /Volumes/"$VOLUME_NAME" --nointeraction || exit 1

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
