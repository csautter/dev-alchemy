packer {
  required_version = ">= 1.12.0"
  required_plugins {
    virtualbox = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/virtualbox"
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

variable "nested_virt" {
  type    = bool
  default = true
}

variable "cpus" {
  type    = number
  default = 2
}

source "virtualbox-iso" "win11" {
  vm_name          = "win11-packer-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  output_directory = "${path.root}/../../../vendor/windows/virtualbox-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"

  iso_url      = var.iso_url
  iso_checksum = "none"

  guest_os_type = "Windows11_64"
  memory        = 4096
  cpus          = var.cpus
  disk_size     = 61440
  nested_virt   = var.nested_virt

  communicator   = "winrm"
  winrm_username = "Administrator"
  winrm_password = "P@ssw0rd!"

  boot_wait = "2s"
  boot_command = [
    "<spacebar>"
  ]

  cd_files = [
    "${path.root}/virtualbox/autounattend.xml"
  ]

  shutdown_command = "shutdown /s /t 10 /f /d p:4:1 /c \"Packer Shutdown\""
  shutdown_timeout = "5m"
}

build {
  sources = ["source.virtualbox-iso.win11"]

  post-processor "vagrant" {
    output              = "${path.root}/../../../cache/windows11/virtualbox-windows11-amd64.box"
    keep_input_artifact = false
    provider_override   = "virtualbox"
    compression_level   = 1
  }
}
