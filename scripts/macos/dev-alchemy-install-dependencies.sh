#!/usr/bin/env bash

set -ex

brew tap hashicorp/tap
brew install hashicorp/tap/packer

brew install --cask utm
brew install qemu

brew install xz
brew install ffmpeg
brew install vncsnapshot