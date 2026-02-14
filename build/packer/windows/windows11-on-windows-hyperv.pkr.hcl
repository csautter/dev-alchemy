packer {
  required_version = ">= 1.12.0"
  required_plugins {
    hyperv = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/hyperv"
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

source "hyperv-iso" "win11" {
  vm_name          = "win11-packer-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  output_directory = "${path.root}/../../../vendor/windows/hyperv-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"

  iso_url      = var.iso_url
  iso_checksum = "none"
  # Ensure that the "Default Switch" exists in Hyper-V.
  # You can check in Hyper-V Manager under "Virtual Switch Manager".
  # If it does not exist, create a new virtual switch named "Default Switch".
  switch_name = "Default Switch"
  memory      = 4096
  cpus        = min(packer.num_cores, 4)
  disk_size   = 61440

  communicator   = "winrm"
  winrm_username = "Administrator"
  winrm_password = "P@ssw0rd!"

  enable_secure_boot = true
  generation         = 2
  enable_tpm         = true

  boot_wait = "500ms"

  # Send multiple keypresses to ensure we catch the "press any key" prompt
  # The prompt typically has a 5-second timeout window
  boot_command = [
    "<spacebar><wait1s>",
    "<spacebar><wait1s>",
    "<spacebar><wait1s>",
    "<spacebar>"
  ]

  # The "autounattend.xml" file is an unattended setup configuration for Windows installation.
  cd_files = [
    "${path.root}/hyperv/autounattend.xml"
  ]

  shutdown_command = "shutdown /s /t 10 /f /d p:4:1 /c \"Packer Shutdown\""
  shutdown_timeout = "5m"
}

build {
  sources = ["source.hyperv-iso.win11"]

  post-processor "vagrant" {
    output              = "${path.root}/../../../cache/windows11/hyperv-windows11-amd64.box"
    keep_input_artifact = false
    provider_override   = "hyperv"
    compression_level   = 1
  }
}
