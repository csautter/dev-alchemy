#!/bin/bash

# This script downloads the latest UTM Guest Tools ISO for macOS hosts
# and saves it to the cache/utm directory.
set -ex
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DATA_DIR="${DEV_ALCHEMY_APP_DATA_DIR:-$HOME/Library/Application Support/dev-alchemy}"
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
