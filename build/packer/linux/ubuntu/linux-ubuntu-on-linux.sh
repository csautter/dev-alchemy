#!/usr/bin/env bash

set -e

script_dir=$(
	cd "$(dirname "$0")"
	pwd -P
)

exec bash "${script_dir}/linux-ubuntu-qemu.sh" "$@"
