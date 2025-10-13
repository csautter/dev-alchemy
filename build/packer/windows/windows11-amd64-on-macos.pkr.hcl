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
  default = "../../../vendor/windows/Win11_25H2_English_x64.iso"
}

# Set to true to run QEMU in headless mode (no GUI)
variable "headless" {
  type    = bool
  default = false
}

source "qemu" "win11" {
  # Apple Silicon host → x86 guest → needs software emulation
  accelerator     = "tcg"
  cpu_model       = "Haswell"
  machine_type    = "q35"
  disk_size       = "64G"
  disk_interface  = "ide"
  format          = "qcow2"
  headless        = var.headless
  iso_url         = var.iso_url
  iso_checksum    = "none"
  cdrom           = var.iso_url
  cdrom_files     = [../../../vendor/utm/utm-guest-tools-latest.iso]
  output_directory = "${path.root}/../../../vendor/windows/qemu-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  display         = "cocoa"
  memory          = "4096"
  cores           = 4
  net_device      = "e1000"

  tpm_device_type = "emulator"

  # The autounattend.xml will be mounted as a virtual floppy drive
  floppy_files = ["${path.root}/qemu-amd64/autounattend.xml"]

  vnc_bind_address = "127.0.0.1"
  vnc_port_min     = 5902
  vnc_port_max     = 5902
  vnc_use_password = true
  vnc_password     = "packer"

  boot_wait = "5s"

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
