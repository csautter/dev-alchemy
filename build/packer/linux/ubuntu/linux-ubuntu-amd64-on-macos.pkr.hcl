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

# Set to true to run QEMU in headless mode (no GUI)
variable "headless" {
  type    = bool
  default = false
}

variable "vnc_port" {
  type    = number
  default = 5901
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
  boot_command     = local.boot_command_server
  left_list        = join("", [for i in range(0, 16) : "<left>"])
  output_directory = "${path.root}/../../../../internal/linux/qemu-ubuntu-${var.ubuntu_type}-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
}

source "qemu" "ubuntu" {
  vm_name          = "linux-ubuntu-${var.ubuntu_type}-packer-amd64"
  headless         = var.headless
  output_directory = local.output_directory
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
    "${path.root}/cloud-init/qemu-${var.ubuntu_type}/meta-data",
    "${path.root}/cloud-init/qemu-${var.ubuntu_type}/user-data"
  ]

  vnc_bind_address = "127.0.0.1"
  vnc_port_min     = var.vnc_port
  vnc_port_max     = var.vnc_port
  vnc_use_password = true
  vnc_password     = "packer"

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

  post-processor "shell-local" {
    inline = [
      "echo 'Exporting QCOW2 image...'",
      "mkdir -p ${path.root}/../../../../internal/linux/linux-ubuntu-${var.ubuntu_type}-qemu-amd64",
      "cp ${local.output_directory}/linux-ubuntu-${var.ubuntu_type}-packer-* ${path.root}/../../../../internal/linux/linux-ubuntu-${var.ubuntu_type}-qemu-amd64/linux-ubuntu-${var.ubuntu_type}-packer.qcow2",
      "echo 'Export completed.'"
    ]
  }
}
