packer {
  required_plugins {
    qemu = {
      version = ">= 1.1.0"
      source  = "github.com/hashicorp/qemu"
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
  ubuntu_iso_url      = "https://releases.ubuntu.com/${var.ubuntu_version}/ubuntu-${var.ubuntu_version}-live-server-amd64.iso"
  ubuntu_iso_checksum = "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b"
  boot_command_server = [
    "e<wait2>",
    "<leftShiftOn><down><down><down><end><leftShiftOff><wait2>",
    "<leftShiftOn><left><left><left><leftShiftOff><wait> autoinstall ds=nocloud ",
    "<wait2>",
    "<f10><wait>",
  ]
  boot_command = local.boot_command_server
  type_name    = var.desktop_version ? "desktop" : "server"
  left_list    = join("", [for i in range(0, 16) : "<left>"])
}

source "qemu" "ubuntu" {
  vm_name          = "linux-ubuntu-${local.type_name}-packer-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  output_directory = "${path.root}/../../../../vendor/linux/hyperv-ubuntu-${local.type_name}-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  iso_url          = local.ubuntu_iso_url
  iso_checksum     = local.ubuntu_iso_checksum
  memory           = 4096
  accelerator      = "tcg"
  cpu_model        = "Skylake-Client"
  machine_type     = "q35"
  cpus             = 1
  disk_size        = "64G"
  disk_interface   = "ide"
  format           = "qcow2"
  display          = "cocoa"
  net_device       = "e1000"

  # Cloud-init seed ISO
  cd_label = "cidata"
  cd_files = [
    "${path.root}/cloud-init/qemu-${local.type_name}/meta-data",
    "${path.root}/cloud-init/qemu-${local.type_name}/user-data"
  ]

  communicator = "ssh"
  ssh_username = "packer"
  ssh_password = "P@ssw0rd!"
  ssh_timeout  = "4h"

  boot_wait = "2s"

  boot_command = local.boot_command

  shutdown_command = "echo 'P@ssw0rd!' | sudo -S shutdown -P now"


  qemuargs = [
    ["-machine", "q35,vmport=off,i8042=off,hpet=off"],
    ["-accel", "tcg,thread=multi,tb-size=1024"],
    ["-smp", "cpus=4,cores=4,sockets=1,threads=1"],
    ["-global", "PIIX4_PM.disable_s3=1"],
    ["-global", "ICH-LPC.disable_s3=1"],
    ["-device", "qemu-xhci"],
    ["-device", "usb-kbd"],
    ["-device", "usb-tablet"],
    ["-device", "usb-mouse"],
  ]
}

build {
  sources = ["source.qemu.ubuntu"]
}
