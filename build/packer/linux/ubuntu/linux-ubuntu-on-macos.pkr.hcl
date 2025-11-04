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

variable "arch" {
  type        = string
  default     = "amd64"
  description = "Target architecture: amd64 or arm64."
  validation {
    condition     = var.arch == "amd64" || var.arch == "arm64"
    error_message = "The variable arch must be either 'amd64' or 'arm64'."
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

variable "iso_url" {
  type    = string
  default = "../../../../vendor/linux/ubuntu-24.04.3-live-server-arm64.iso"
}

variable "is_ci" {
  type    = bool
  default = env("CI") == "true"
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
  iso_url = var.arch == "amd64" ? "https://releases.ubuntu.com/${var.ubuntu_version}/ubuntu-${var.ubuntu_version}-live-${var.ubuntu_type}-amd64.iso" : "${path.root}/${var.iso_url}"

  ubuntu_iso_checksum = var.arch == "amd64" ? "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b" : "none"

  boot_command = {
    "amd64" = [
      "e<wait2>",
      "<leftShiftOn><down><down><down><end><leftShiftOff><wait2>",
      "<leftShiftOn><left><left><left><leftShiftOff><wait> autoinstall ds=nocloud ",
      "<wait2>",
      "<f10><wait>",
    ]
    "arm64" = [
      "e<wait2>",
      "<down><down><down><end><wait2>",
      "${local.left_list}<wait> autoinstall ds=nocloud ",
      "<wait2>",
      "<f10><wait>",
    ]
  }
  qemu_args = {
    "amd64" = [
      ["-machine", "q35,vmport=off,i8042=off,hpet=off"],
      ["-accel", "tcg,thread=multi,tb-size=1024"],
      ["-smp", "cpus=4,cores=4,sockets=1,threads=1"],
      ["-global", "PIIX4_PM.disable_s3=1"],
      ["-global", "ICH-LPC.disable_s3=1"],
      ["-device", "qemu-xhci"],
      ["-device", "usb-kbd"],
      ["-device", "usb-tablet"],
      ["-device", "usb-mouse"],
      ["-k", "de"],
    ]
    "arm64" = [
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
      ["-boot", "order=d,menu=on"],
      ["-k", "de"],
    ]
  }
  left_list        = join("", [for i in range(0, 16) : "<left>"])
  output_directory = "${path.root}/../../../../internal/linux/qemu-ubuntu-${var.ubuntu_type}-out-${var.arch}"
}

source "qemu" "ubuntu" {
  qemu_binary      = var.arch == "amd64" ? "qemu-system-x86_64" : "qemu-system-aarch64"
  vm_name          = "linux-ubuntu-${var.ubuntu_type}-packer-${var.arch}"
  headless         = var.headless
  output_directory = local.output_directory
  iso_url          = local.iso_url
  iso_checksum     = local.ubuntu_iso_checksum
  memory           = 4096
  cpu_model        = var.arch == "amd64" ? "Skylake-Client" : "max" # overwritten by qemu arg for arm64
  disk_size        = "64G"                                          # overwritten by qemu arg for arm64
  disk_interface   = "ide"                                          # overwritten by qemu arg for arm64
  format           = "qcow2"                                        # overwritten by qemu arg for arm64
  display          = "cocoa"
  net_device       = var.arch == "amd64" ? "e1000" : "virtio-net-pci"

  # Cloud-init seed ISO
  # overwritten by qemu arg for arm64
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

  boot_wait = var.arch == "amd64" ? "2s" : "10s"

  boot_command = local.boot_command[var.arch]

  shutdown_command = "echo 'P@ssw0rd!' | sudo -S shutdown -P now"

  qemuargs = local.qemu_args[var.arch]
}

build {
  sources = ["source.qemu.ubuntu"]

  post-processor "shell-local" {
    inline = var.arch == "amd64" ? [
      "echo 'Exporting QCOW2 image...'",
      "mkdir -p ${path.root}/../../../../internal/linux/linux-ubuntu-${var.ubuntu_type}-qemu-${var.arch}",
      "cp ${local.output_directory}/linux-ubuntu-${var.ubuntu_type}-packer-* ${path.root}/../../../../internal/linux/linux-ubuntu-${var.ubuntu_type}-qemu-${var.arch}/linux-ubuntu-${var.ubuntu_type}-packer.qcow2",
      "echo 'Export completed.'"
      ] : [
      "echo 'No export needed for arm64 architecture.'"
    ]
  }
}
