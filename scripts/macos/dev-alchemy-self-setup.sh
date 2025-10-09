#!/usr/bin/env bash

set -ex

brew tap hashicorp/tap
brew install hashicorp/tap/packer
brew install hashicorp/tap/hashicorp-vagrant

brew install qemu
brew install --cask utm
brew install libvirt
vagrant plugin install vagrant-libvirt

brew install --cask powershell

brew install xz