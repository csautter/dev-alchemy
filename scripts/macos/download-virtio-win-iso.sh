#!/bin/bash

set -ex

APP_DATA_DIR="${DEV_ALCHEMY_APP_DATA_DIR:-$HOME/Library/Application Support/dev-alchemy}"
CACHE_DIR="${DEV_ALCHEMY_CACHE_DIR:-$APP_DATA_DIR/cache}"
export DEV_ALCHEMY_APP_DATA_DIR="$APP_DATA_DIR"
export DEV_ALCHEMY_CACHE_DIR="$CACHE_DIR"

# renovate: datasource=custom.virtio-win depName=virtio-win versioning=loose
VIRTIO_WIN_VERSION="0.1.266-1"
VIRTIO_WIN_FILE_VERSION="${VIRTIO_WIN_VERSION%-*}"

if [ ! -f "$CACHE_DIR/windows/virtio-win.iso" ]; then
	echo "Downloading virtio-win.iso"
	mkdir -p "$CACHE_DIR/windows"
	curl --progress-bar -L -o "$CACHE_DIR/windows/virtio-win.iso" "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-${VIRTIO_WIN_VERSION}/virtio-win-${VIRTIO_WIN_FILE_VERSION}.iso"
else
	echo "virtio-win.iso already exists, skipping download"
	exit 0
fi
