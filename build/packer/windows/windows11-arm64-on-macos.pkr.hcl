# Check out this guide for more information on setting up Windows on ARM64 using QEMU on macOS:
# https://linaro.atlassian.net/wiki/spaces/WOAR/pages/28914909194/windows-arm64+VM+using+qemu-system

packer {
  required_plugins {
    qemu = {
      version = ">= 1.1.0"
      source  = "github.com/hashicorp/qemu"
    }
    vagrant = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/vagrant"
    }
  }
}

variable "iso_url" {
  type    = string
  default = "../../../vendor/windows/Win11_25H2_English_arm64.iso"
}

source "qemu" "win11" {
  # Apple Silicon host → x86 guest → needs software emulation
  accelerator     = "hvf"
  cpu_model       = "host"
  machine_type    = "virt"
  qemu_binary    = "qemu-system-aarch64"
  disk_size       = "64G"
  disk_interface  = "virtio"
  format          = "qcow2"
  # you can enable headless mode by uncommenting the following line
  # headless        = true
  iso_url         = var.iso_url
  iso_checksum    = "none"
  #cd_files     = ["${path.root}/../../../vendor/utm/utm-guest-tools-latest.iso"]
  #cdrom_interface = "ide"
  output_directory = "${path.root}/../../../vendor/windows/qemu-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  display         = "cocoa"
  memory          = "4096"
  cores           = 4
  net_device      = "e1000"

  tpm_device_type = "tpm-tis-device"

  boot_wait = "5s"
  boot_command = [
    "<spacebar>",
    "<wait3>",
    "<spacebar>",
    "<wait3>",
    "<spacebar>",
  ]

  qemuargs = [
    ["-bios", "${path.root}/../../../vendor/qemu-uefi/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"],
    ["-device","ramfb"],
    ["-device","qemu-xhci"],
    ["-device","usb-kbd"],
    ["-device","usb-tablet"],
    ["-device", "usb-storage,drive=install"],
    ["-drive", "if=none,id=install,format=raw,media=cdrom,file=${var.iso_url}"],
    ["-device", "usb-storage,drive=autounattend"],
    ["-drive", "if=none,id=autounattend,format=raw,media=cdrom,file=${path.root}/../../../vendor/windows/autounattend.iso"],
    ["-device", "usb-storage,drive=virtio-drivers"],
    ["-drive", "if=none,id=virtio-drivers,format=raw,media=cdrom,file=${path.root}/../../../vendor/windows/virtio-win.iso"],
    ["-boot", "order=d,menu=on"]
  ]

  # The autounattend.xml will be mounted as a virtual floppy drive
  cd_files = ["${path.root}/qemu/autounattend.xml"]

  communicator   = "winrm"
  winrm_username = "Administrator"
  winrm_password = "P@ssw0rd!"
  winrm_timeout  = "6h"

  shutdown_command = "shutdown /s /t 10 /f /d p:4:1 /c \"Packer Shutdown\""
  shutdown_timeout = "5m"
}

build {
  sources = ["source.qemu.win11"]

  # This provisioner creates C:\packer.txt to verify that the VM was successfully provisioned by Packer.
  /*
  provisioner "powershell" {
    inline = [
      "Write-Output 'Running inside Windows VM...'",
      "New-Item -Path C:\\packer.txt -ItemType File -Force",
      "Write-Output 'Created C:\\packer.txt file.'",
      # delete the file to keep the image clean
      "Remove-Item -Path C:\\packer.txt -Force"
    ]
  }*/

  /*
  post-processor "vagrant" {
    output = "${path.root}/../../../vendor/windows/win11-qemu.box"
    keep_input_artifact = false
    compression_level = 0
  }
  */
}
