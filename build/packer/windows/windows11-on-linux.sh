#!/usr/bin/env bash

set -e

script_dir=$(
	cd "$(dirname "$0")" || exit 1
	pwd -P
)

exec bash "${script_dir}/windows11-qemu.sh" "$@"
