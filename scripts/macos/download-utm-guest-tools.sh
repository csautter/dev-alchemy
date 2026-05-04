#!/bin/bash

# This script downloads the latest UTM Guest Tools ISO for QEMU Windows guests
# and saves it to the cache/utm directory.
set -ex
host_os="$(uname -s)"
if [ "$host_os" = "Darwin" ]; then
	DEFAULT_APP_DATA_DIR="$HOME/Library/Application Support/dev-alchemy"
else
	DEFAULT_APP_DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/dev-alchemy"
fi

APP_DATA_DIR="${DEV_ALCHEMY_APP_DATA_DIR:-$DEFAULT_APP_DATA_DIR}"
CACHE_DIR="${DEV_ALCHEMY_CACHE_DIR:-$APP_DATA_DIR/cache}"
export DEV_ALCHEMY_APP_DATA_DIR="$APP_DATA_DIR"
export DEV_ALCHEMY_CACHE_DIR="$CACHE_DIR"
OUTPUT_DIR="$CACHE_DIR/utm"
OUTPUT_PATH="$OUTPUT_DIR/utm-guest-tools-latest.iso"

if [ -f "$OUTPUT_PATH" ]; then
	echo "UTM Guest Tools ISO already exists at $OUTPUT_PATH, skipping download."
	exit 0
fi

mkdir -p "$OUTPUT_DIR"
curl --progress-bar -L -o "$OUTPUT_PATH" "https://getutm.app/downloads/utm-guest-tools-latest.iso"
