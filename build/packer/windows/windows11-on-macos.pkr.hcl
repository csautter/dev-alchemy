packer {
  required_plugins {
    qemu = {
      version = ">= 1.1.0"
      source  = "github.com/hashicorp/qemu"
    }
  }
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

variable "iso_url" {
  type        = string
  default     = ""
  description = "Path to Windows 11 ISO. If empty, will be set by arch."
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

locals {
  cache_directory = "${path.root}/../../../../cache"
  win11_default_iso = {
    amd64 = "../../../vendor/windows/win11_25h2_english_amd64.iso"
    arm64 = "../../../vendor/windows/Win11_25H2_English_arm64.iso"
  }
  win11_iso         = var.iso_url != "" ? var.iso_url : local.win11_default_iso[var.arch]
  win11_qcow2       = "${local.cache_directory}/windows/qemu-windows11-${var.arch}.qcow2"
  win11_guest_tools = "${path.root}/../../../vendor/utm/utm-guest-tools-latest.iso"
  win11_virtio_iso  = "${path.root}/../../../vendor/windows/virtio-win.iso"
  win11_uefi_bios   = "${path.root}/../../../vendor/qemu-uefi/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"
  qemu_args = {
    "amd64" = [
      ["-device", "qemu-xhci,id=usb"],
      ["-device", "usb-storage,drive=install,removable=true,bootindex=0"],
      ["-drive", "if=none,id=install,format=raw,media=cdrom,file=${local.win11_iso},readonly=true"],
      ["-device", "usb-storage,drive=utm-tools,removable=true,bootindex=2"],
      ["-drive", "if=none,id=utm-tools,format=raw,media=cdrom,file=${local.win11_guest_tools},readonly=true"],
      ["-device", "ide-hd,drive=ide0,bootindex=1"],
      ["-drive", "if=none,media=disk,id=ide0,format=qcow2,file.filename=${local.win11_qcow2},discard=unmap,detect-zeroes=unmap"],
      ["-boot", "order=c,menu=on"],
      ["-k", "de"]
    ],
    "arm64" = [
      ["-accel", var.is_ci ? "tcg,thread=multi,tb-size=512" : "hvf"],
      ["-machine", "virt,highmem=on"],
      # max cpu model is best choice here:
      # pmu=off is causing issues while loading windows 11 arm64 setup
      ["-cpu", var.is_ci ? "max,sve=off,pauth-impdef=on" : "host"],
      # setting a specific cpu model leads to issues, therefore using max above
      # ["-cpu", var.is_ci ? "cortex-a72" : "host"],
      ["-bios", "${path.root}/../../../vendor/qemu-uefi/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"],
      ["-device", "ramfb"],
      ["-device", "qemu-xhci"],
      ["-device", "usb-kbd"],
      ["-device", "usb-tablet"],
      ["-device", "usb-storage,drive=install,removable=true,bootindex=0"],
      ["-drive", "if=none,id=install,format=raw,media=cdrom,file=${local.win11_iso},readonly=true"],
      ["-device", "usb-storage,drive=virtio-drivers,removable=true,bootindex=2"],
      ["-drive", "if=none,id=virtio-drivers,format=raw,media=cdrom,file=${path.root}/../../../vendor/windows/virtio-win.iso,readonly=true"],
      ["-device", "usb-storage,drive=utm-tools,removable=true,bootindex=3"],
      ["-drive", "if=none,id=utm-tools,format=raw,media=cdrom,file=${path.root}/../../../vendor/utm/utm-guest-tools-latest.iso,readonly=true"],
      ["-device", "nvme,drive=nvme0,serial=deadbeef,bootindex=1"],
      ["-drive", "if=none,media=disk,id=nvme0,format=qcow2,file.filename=${local.cache_directory}/windows/qemu-windows11-arm64.qcow2,discard=unmap,detect-zeroes=unmap"],
      ["-boot", "order=c,menu=on"],
      ["-k", "de"]
    ]
  }
}

source "qemu" "win11" {
  vm_name = "windows11-packer-${var.arch}"

  headless         = var.headless
  iso_url          = var.iso_url != "" ? var.iso_url : local.win11_iso[var.arch]
  iso_checksum     = "none"
  output_directory = "${local.cache_directory}/windows/qemu-windows-out-${var.arch}"
  display          = "cocoa"
  memory           = "4096"
  cores            = var.is_ci ? 3 : 4
  vnc_bind_address = "127.0.0.1"
  vnc_port_min     = var.vnc_port
  vnc_port_max     = var.vnc_port
  vnc_use_password = true
  vnc_password     = "packer"
  boot_wait        = "5s"
  boot_command = var.arch == "arm64" ? [
    "<spacebar>",
    "<wait3>",
    "<spacebar>",
    "<wait3>",
    "<spacebar>",
  ] : null
  communicator            = "winrm"
  pause_before_connecting = var.is_ci ? "10m" : "1m"
  winrm_username          = "Administrator"
  winrm_password          = "P@ssw0rd!"
  winrm_timeout           = var.is_ci ? "4h" : "5h"
  shutdown_command        = "shutdown /s /t 60 /f /d p:4:1 /c \"Packer Shutdown\""
  shutdown_timeout        = "10m"

  # Arch-specific config
  machine_type   = var.arch == "amd64" ? "q35" : null
  accelerator    = var.arch == "amd64" ? "tcg" : null
  cpu_model      = var.arch == "amd64" ? "Haswell" : null
  qemu_binary    = var.arch == "arm64" ? "qemu-system-aarch64" : "qemu-system-x86_64"
  disk_size      = "64G"
  disk_interface = var.arch == "amd64" ? "ide" : null
  format         = "qcow2"
  net_device     = var.arch == "amd64" ? "e1000" : "virtio-net-pci"

  tpm_device_type = var.arch == "amd64" ? "emulator" : null

  floppy_files = var.arch == "amd64" ? ["${path.root}/qemu-amd64/autounattend.xml"] : null

  qemuargs = local.qemu_args[var.arch]
}


build {
  sources = ["source.qemu.win11"]
}
