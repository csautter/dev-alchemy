#!/usr/bin/env bash

set -ex

install_go="false"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--with-go)
		install_go="true"
		shift
		;;
	*)
		echo "Unknown option: $1" >&2
		exit 1
		;;
	esac
done

brew tap hashicorp/tap
brew install hashicorp/tap/packer

brew install --cask utm
brew install qemu

brew install xz
brew install ffmpeg
brew install vncsnapshot
brew install xorriso
brew install ansible

if [[ "${install_go}" == "true" ]]; then
	brew install go
fi
