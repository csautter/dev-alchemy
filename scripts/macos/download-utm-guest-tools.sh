#!/bin/bash

# This script downloads the latest UTM Guest Tools ISO for macOS hosts
# and saves it to the vendor/utm directory.
set -ex
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="$SCRIPT_DIR/../../vendor/utm"
OUTPUT_PATH="$OUTPUT_DIR/utm-guest-tools-latest.iso"

mkdir -p "$OUTPUT_DIR"
curl --progress-bar -L -o "$OUTPUT_PATH" "https://getutm.app/downloads/utm-guest-tools-latest.iso"