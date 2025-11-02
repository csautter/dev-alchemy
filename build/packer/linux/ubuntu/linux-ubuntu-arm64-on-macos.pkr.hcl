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

variable "ubuntu_type" {
  type        = string
  default     = "server"
  description = "The type of Ubuntu image to build (server or desktop)."
  validation {
    condition     = var.ubuntu_type == "server" || var.ubuntu_type == "desktop"
    error_message = "The variable ubuntu_type must be either 'server' or 'desktop'."
  }
}

# Set to true to run QEMU in headless mode (no GUI)
variable "headless" {
  type    = bool
  default = false
}

variable "vnc_port" {
  type    = number
  default = 5901
}

variable "is_ci" {
  type    = bool
  default = env("CI") == "true"
}

variable "iso_url" {
  type    = string
  default = "../../../../vendor/linux/ubuntu-24.04.3-live-server-arm64.iso"
}

locals {
  iso_url = "${path.root}/${var.iso_url}"
  boot_command_server = [
    "e<wait2>",
    "<down><down><down><end><wait2>",
    "${local.left_list}<wait> autoinstall ds=nocloud ",
    "<wait2>",
    "<f10><wait>",
  ]
  boot_command     = local.boot_command_server
  left_list        = join("", [for i in range(0, 16) : "<left>"])
  output_directory = "${path.root}/../../../../internal/linux/qemu-ubuntu-${var.ubuntu_type}-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
}

source "qemu" "ubuntu" {
  qemu_binary      = "qemu-system-aarch64"
  headless         = var.headless
  vm_name          = "linux-ubuntu-${var.ubuntu_type}-packer-arm64"
  output_directory = local.output_directory
  memory           = 4096
  cores            = var.is_ci ? 3 : 4
  display          = "cocoa"
  net_device       = "virtio-net-pci"

  iso_url      = local.iso_url
  iso_checksum = "none"

  vnc_bind_address = "127.0.0.1"
  vnc_port_min     = var.vnc_port
  vnc_port_max     = var.vnc_port
  vnc_use_password = true
  vnc_password     = "packer"

  # Cloud-init seed ISO
  cd_label = "cidata"
  cd_files = [
    "${path.root}/cloud-init/qemu-${var.ubuntu_type}/meta-data",
    "${path.root}/cloud-init/qemu-${var.ubuntu_type}/user-data"
  ]

  communicator = "ssh"
  ssh_username = "packer"
  ssh_password = "P@ssw0rd!"
  ssh_timeout  = "4h"

  boot_wait = "10s"

  boot_command = local.boot_command

  shutdown_command = "echo 'P@ssw0rd!' | sudo -S shutdown -P now"

  qemuargs = [
    ["-accel", var.is_ci ? "tcg,thread=multi,tb-size=512" : "hvf"],
    ["-machine", "virt,highmem=on"],
    ["-cpu", var.is_ci ? "max,sve=off,pauth-impdef=on" : "host"],
    ["-bios", "${path.root}/../../../../vendor/qemu-uefi/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"],
    ["-device", "ramfb"],
    ["-smp", "cpus=4,cores=4,sockets=1,threads=1"],
    ["-global", "PIIX4_PM.disable_s3=1"],
    ["-global", "ICH-LPC.disable_s3=1"],
    ["-device", "qemu-xhci"],
    ["-device", "usb-kbd"],
    ["-device", "usb-tablet"],
    ["-device", "usb-mouse"],
    # Installation ISO
    ["-device", "virtio-blk-pci,drive=cdrom,bootindex=1"],
    ["-drive", "if=none,id=cdrom,media=cdrom,file=${local.iso_url},readonly=true"],
    # Main disk
    ["-device", "virtio-blk-pci,drive=disk,serial=deadbeef,bootindex=0"],
    ["-drive", "if=none,media=disk,id=disk,format=qcow2,file.filename=${path.root}/../../../../internal/linux/linux-ubuntu-${var.ubuntu_type}-qemu-arm64/linux-ubuntu-${var.ubuntu_type}-packer.qcow2,discard=unmap,detect-zeroes=unmap"],
    # Cloud-init seed ISO
    ["-drive", "if=none,id=cidata,format=raw,file=${path.root}/cloud-init/qemu-${var.ubuntu_type}/cidata.iso,readonly=true"],
    ["-device", "virtio-blk-pci,drive=cidata"],
    ["-boot", "order=d,menu=on"]
  ]
}

build {
  sources = [
    "source.qemu.ubuntu"
  ]
}
