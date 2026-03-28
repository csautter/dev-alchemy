#!/bin/bash

set -ex

SCRIPT_DIR=$(
	cd $(dirname $0)
	pwd
)

APP_DATA_DIR="${DEV_ALCHEMY_APP_DATA_DIR:-$HOME/Library/Application Support/dev-alchemy}"
CACHE_DIR="${DEV_ALCHEMY_CACHE_DIR:-$APP_DATA_DIR/cache}"
export DEV_ALCHEMY_APP_DATA_DIR="$APP_DATA_DIR"
export DEV_ALCHEMY_CACHE_DIR="$CACHE_DIR"

if [ ! -f "$CACHE_DIR/windows/virtio-win.iso" ]; then
	echo "Downloading virtio-win.iso"
	mkdir -p "$CACHE_DIR/windows"
	curl --progress-bar -L -o "$CACHE_DIR/windows/virtio-win.iso" https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.266-1/virtio-win-0.1.266.iso
else
	echo "virtio-win.iso already exists, skipping download"
	exit 0
fi
