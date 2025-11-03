packer {
  required_plugins {
    qemu = {
      version = ">= 1.1.0"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

variable "iso_url" {
  type    = string
  default = "../../../vendor/windows/win11_25h2_english_amd64.iso"
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

source "qemu" "win11" {
  vm_name = "windows11-packer-amd64"
  # Apple Silicon host → x86 guest → needs software emulation
  accelerator      = "tcg"
  cpu_model        = "Haswell"
  machine_type     = "q35"
  disk_size        = "64G"
  disk_interface   = "ide"
  format           = "qcow2"
  headless         = var.headless
  iso_url          = var.iso_url
  iso_checksum     = "none"
  output_directory = "${path.root}/../../../vendor/windows/qemu-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  display          = "cocoa"
  memory           = "4096"
  # Github Actions macOS runners have 3 CPU cores, so limit to 3 when running in CI
  # https://docs.github.com/en/actions/reference/runners/github-hosted-runners#standard-github-hosted-runners-for-public-repositories
  cores      = var.is_ci ? 3 : 4
  net_device = "e1000"

  tpm_device_type = "emulator"

  # The autounattend.xml will be mounted as a virtual floppy drive
  floppy_files = ["${path.root}/qemu-amd64/autounattend.xml"]

  vnc_bind_address = "127.0.0.1"
  vnc_port_min     = var.vnc_port
  vnc_port_max     = var.vnc_port
  vnc_use_password = true
  vnc_password     = "packer"

  boot_wait = "5s"

  communicator            = "winrm"
  pause_before_connecting = var.is_ci ? "10m" : "1m"
  winrm_username          = "Administrator"
  winrm_password          = "P@ssw0rd!"
  winrm_timeout           = "5h"

  shutdown_command = "shutdown /s /t 60 /f /d p:4:1 /c \"Packer Shutdown\""
  shutdown_timeout = "10m"

  qemuargs = [
    ["-device", "qemu-xhci,id=usb"],
    ["-device", "usb-storage,drive=install,removable=true,bootindex=0"],
    ["-drive", "if=none,id=install,format=raw,media=cdrom,file=${var.iso_url},readonly=true"],
    ["-device", "usb-storage,drive=utm-tools,removable=true,bootindex=2"],
    ["-drive", "if=none,id=utm-tools,format=raw,media=cdrom,file=${path.root}/../../../vendor/utm/utm-guest-tools-latest.iso,readonly=true"],
    ["-device", "ide-hd,drive=ide0,bootindex=1"],
    ["-drive", "if=none,media=disk,id=ide0,format=qcow2,file.filename=${path.root}/../../../internal/windows/qemu-windows11-amd64.qcow2,discard=unmap,detect-zeroes=unmap"],
    ["-boot", "order=c,menu=on"],
    ["-k", "de"]
  ]
}

build {
  sources = ["source.qemu.win11"]
}
