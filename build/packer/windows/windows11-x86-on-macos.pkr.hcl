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
  default = "../../../vendor/windows/Win11_25H2_English_x64.iso"
}

source "qemu" "win11" {
  # Apple Silicon host → x86 guest → needs software emulation
  accelerator     = "tcg"
  cpu_model       = "Skylake-Client"
  machine_type    = "q35"
  disk_size       = "64G"
  disk_interface  = "ide"
  format          = "qcow2"
  #headless        = true
  iso_url         = var.iso_url
  iso_checksum    = "none"
  output_directory   = "${path.root}/../../../vendor/windows/hyperv-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  #use_default_display = true
  display         = "cocoa"
  memory          = "4096"
  cores            = 10

  #qemuargs = [
  #  ["--device", "ich9-ahci,id=sata"],
  #]

  # UEFI BIOS (recommended for Win11, but not strictly required for installation)
  # Windows 11 officially requires UEFI for Secure Boot and TPM, but it can sometimes be installed in legacy BIOS mode,
  # especially if hardware checks are bypassed. UTM may use legacy BIOS by default for x86 VMs.
  # Uncomment the following lines to enable UEFI if needed:
  # efi_boot = true
  # efi_firmware_code = "/Applications/UTM.app/Contents/Resources/qemu/edk2-x86_64-code.fd"
  # efi_firmware_vars = "/Applications/UTM.app/Contents/Resources/qemu/edk2-x86_64-code.fd"

  tpm_device_type = "emulator"

  # The autounattend.xml will be mounted as a virtual floppy drive
  floppy_files = ["${path.root}/qemu/autounattend.xml"]

  # Boot configuration
  boot_wait = "5s"
  #boot_command = [
  #  "<esc><wait>",
  #  "<f12><wait>",
  #  "1<enter>"
  #]

  communicator   = "winrm"
  winrm_username = "Administrator"
  winrm_password = "P@ssw0rd!"

  shutdown_command = "shutdown /s /t 10 /f /d p:4:1 /c \"Packer Shutdown\""
  shutdown_timeout = "5m"
}

build {
  sources = ["source.qemu.win11"]

  provisioner "shell-local" {
    inline = ["echo 'Windows 11 x86 build finished successfully. Image in output-windows11/disk.qcow2'"]
  }

/*
  post-processor "vagrant" {
    output = "${path.root}/../../../vendor/windows/win11-hyperv.box"
    keep_input_artifact = false
    provider_override = "qemu"
    compression_level = 1
  }
  */
}
