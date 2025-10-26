packer {
  required_plugins {
    hyperv = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/hyperv"
    }
  }
}

variable "ubuntu_version" {
  type    = string
  default = "24.04.3"
}

variable "desktop_version" {
  type    = bool
  default = false
}

locals {
  ubuntu_iso_url      = var.desktop_version ? "https://releases.ubuntu.com/${var.ubuntu_version}/ubuntu-${var.ubuntu_version}-desktop-amd64.iso" : "https://releases.ubuntu.com/${var.ubuntu_version}/ubuntu-${var.ubuntu_version}-live-server-amd64.iso"
  ubuntu_iso_checksum = var.desktop_version ? "sha256:faabcf33ae53976d2b8207a001ff32f4e5daae013505ac7188c9ea63988f8328" : "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b"
  boot_command_server = [
    "e<wait2>",
    "<leftShiftOn><down><down><down><end><leftShiftOff><wait2>",
    "<leftShiftOn><left><left><left><leftShiftOff><wait> autoinstall ds=nocloud ",
    "<wait2>",
    "<f10><wait>",
  ]
  boot_command_desktop = [
    "e<wait2>",
    "<leftShiftOn><down><down><down><end><leftShiftOff><wait2>",
    "<leftShiftOn>${local.left_list}<leftShiftOff><wait> autoinstall ds=nocloud ",
    "<wait2>",
    "<f10><wait>",
  ]
  boot_command = var.desktop_version ? local.boot_command_desktop : local.boot_command_server
  left_list    = join("", [for i in range(0, 16) : "<left>"])
}

source "hyperv-iso" "ubuntu" {
  vm_name            = "linux-ubuntu-packer-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  output_directory   = "${path.root}/../../../../vendor/linux/hyperv-ubuntu-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  iso_url            = local.ubuntu_iso_url
  iso_checksum       = local.ubuntu_iso_checksum
  generation         = 2
  memory             = 4096
  cpus               = 8
  disk_size          = 20000
  switch_name        = "Default Switch"
  enable_secure_boot = false

  # Cloud-init seed ISO
  cd_label = "cidata"
  cd_files = [
    "${path.root}/cloud-init/meta-data",
    "${path.root}/cloud-init/user-data"
  ]

  communicator = "ssh"
  ssh_username = "packer"
  ssh_password = "P@ssw0rd!"
  ssh_timeout  = "60m"

  boot_wait = "2s"

  boot_command = local.boot_command

  shutdown_command = "echo 'P@ssw0rd!' | sudo -S shutdown -P now"
}

build {
  sources = ["source.hyperv-iso.ubuntu"]

  post-processor "vagrant" {
    output              = "${path.root}/../../../../internal/vagrant/linux-ubuntu-hyperv${var.desktop_version ? "-desktop" : "-server"}.box"
    keep_input_artifact = false
    provider_override   = "hyperv"
    compression_level   = 1
  }
}
