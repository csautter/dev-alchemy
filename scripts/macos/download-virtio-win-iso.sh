#!/bin/bash

set -ex

SCRIPT_DIR=$(
	cd $(dirname $0)
	pwd
)

if [ ! -f $SCRIPT_DIR/../../vendor/windows/virtio-win.iso ]; then
	echo "Downloading virtio-win.iso"
	curl --progress-bar -L -o $SCRIPT_DIR/../../vendor/windows/virtio-win.iso https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.266-1/virtio-win-0.1.266.iso
else
	echo "virtio-win.iso already exists, skipping download"
	exit 0
fi
