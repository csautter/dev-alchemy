# Check out this guide for more information on setting up Windows on ARM64 using QEMU on macOS:
# https://linaro.atlassian.net/wiki/spaces/WOAR/pages/28914909194/windows-arm64+VM+using+qemu-system

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
  default = "../../../vendor/windows/Win11_25H2_English_arm64.iso"
}

# Set to true to run QEMU in headless mode (no GUI)
variable "headless" {
  type    = bool
  default = false
}

variable "is_ci" {
  type    = bool
  default = env("CI") == "true"
}

source "qemu" "win11" {
  qemu_binary      = "qemu-system-aarch64"
  headless         = var.headless
  iso_url          = var.iso_url
  iso_checksum     = "none"
  output_directory = "${path.root}/../../../vendor/windows/qemu-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  display          = "cocoa"
  memory           = "4096"
  # Github Actions macOS runners have 3 CPU cores, so limit to 3 when running in CI
  # https://docs.github.com/en/actions/reference/runners/github-hosted-runners#standard-github-hosted-runners-for-public-repositories
  cores      = var.is_ci ? 3 : 4
  net_device = "virtio-net-pci"

  vnc_bind_address = "127.0.0.1"
  vnc_port_min     = 5901
  vnc_port_max     = 5901
  vnc_use_password = true
  vnc_password     = "packer"

  boot_wait = "5s"
  boot_command = [
    "<spacebar>",
    "<wait3>",
    "<spacebar>",
    "<wait3>",
    "<spacebar>",
  ]

  qemuargs = [
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
    ["-drive", "if=none,id=install,format=raw,media=cdrom,file=${var.iso_url},readonly=true"],
    ["-device", "usb-storage,drive=autounattend,removable=true,bootindex=2"],
    ["-drive", "if=none,id=autounattend,format=raw,media=cdrom,file=${path.root}/../../../vendor/windows/autounattend.iso,readonly=true"],
    ["-device", "usb-storage,drive=virtio-drivers,removable=true,bootindex=3"],
    ["-drive", "if=none,id=virtio-drivers,format=raw,media=cdrom,file=${path.root}/../../../vendor/windows/virtio-win.iso,readonly=true"],
    ["-device", "nvme,drive=nvme0,serial=deadbeef,bootindex=1"],
    ["-drive", "if=none,media=disk,id=nvme0,format=qcow2,file.filename=${path.root}/../../../vendor/windows/qemu-windows11-arm64.qcow2,discard=unmap,detect-zeroes=unmap"],
    ["-boot", "order=c,menu=on"]
  ]

  cd_files = ["${path.root}/qemu/autounattend.xml"]

  communicator   = "winrm"
  winrm_username = "Administrator"
  winrm_password = "P@ssw0rd!"
  winrm_timeout  = var.is_ci ? "5h" : "1h"

  shutdown_command = "shutdown /s /t 10 /f /d p:4:1 /c \"Packer Shutdown\""
  shutdown_timeout = "5m"
}

build {
  sources = ["source.qemu.win11"]

  # This provisioner creates C:\packer.txt to verify that the VM was successfully provisioned by Packer.
  provisioner "powershell" {
    inline = [
      "Write-Output 'Running inside Windows VM...'",
      "New-Item -Path C:\\packer.txt -ItemType File -Force",
      "Write-Output 'Created C:\\packer.txt file.'",
      # delete the file to keep the image clean
      "Remove-Item -Path C:\\packer.txt -Force"
    ]
  }
}
