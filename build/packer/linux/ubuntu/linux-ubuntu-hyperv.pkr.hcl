packer {
  required_plugins {
    hyperv = {
      version = ">= 1.1.5"
      source  = "github.com/hashicorp/hyperv"
    }
  }
}

variable "ubuntu_version" {
  type    = string
  default = "24.04.3"
}

variable "iso_url" {
  type        = string
  default     = ""
  description = "Optional local path or URL for the Ubuntu live-server ISO. If empty, the official release URL is used."
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

variable "cpus" {
  type    = number
  default = 4
}

variable "memory" {
  type        = number
  default     = 4096
  description = "Memory in MB to allocate to the VM."
}

variable "switch_name" {
  type    = string
  default = "Default Switch"
}

variable "cache_dir" {
  type        = string
  default     = env("DEV_ALCHEMY_CACHE_DIR")
  description = "Managed cache directory outside the repository."
  validation {
    condition     = var.cache_dir != ""
    error_message = "The cache_dir variable must be set, typically via DEV_ALCHEMY_CACHE_DIR."
  }
}

locals {
  default_ubuntu_iso_url = "https://releases.ubuntu.com/${var.ubuntu_version}/ubuntu-${var.ubuntu_version}-live-server-amd64.iso"
  effective_iso_url      = var.iso_url != "" ? var.iso_url : local.default_ubuntu_iso_url
  ubuntu_iso_checksum    = "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b"
  boot_command = [
    "e<wait2>",
    "<leftShiftOn><down><down><down><end><leftShiftOff><wait2>",
    "<leftShiftOn><left><left><left><leftShiftOff><wait> autoinstall ds=nocloud ",
    "<wait2>",
    "<f10><wait>",
  ]
  output_directory = "${var.cache_dir}/linux/hyperv-ubuntu-${var.ubuntu_type}-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  box_output       = "${var.cache_dir}/ubuntu/hyperv-ubuntu-${var.ubuntu_type}-amd64.box"
}

source "hyperv-iso" "ubuntu" {
  vm_name            = "linux-ubuntu-${var.ubuntu_type}-packer-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  output_directory   = local.output_directory
  iso_url            = local.effective_iso_url
  iso_checksum       = local.ubuntu_iso_checksum
  generation         = 2
  memory             = var.memory
  cpus               = var.cpus
  disk_size          = 64000
  switch_name        = var.switch_name
  enable_secure_boot = false

  cd_label = "cidata"
  cd_files = [
    # Hyper-V builds use dedicated cloud-init content, split by server/desktop.
    "${path.root}/cloud-init/hyperv-${var.ubuntu_type}/meta-data",
    "${path.root}/cloud-init/hyperv-${var.ubuntu_type}/user-data"
  ]

  communicator = "ssh"
  ssh_username = "packer"
  ssh_password = "P@ssw0rd!"
  ssh_timeout  = "4h"

  boot_wait    = "2s"
  boot_command = local.boot_command

  shutdown_command = "echo 'P@ssw0rd!' | sudo -S shutdown -P now"
}

build {
  sources = ["source.hyperv-iso.ubuntu"]

  post-processor "vagrant" {
    output              = local.box_output
    keep_input_artifact = false
    provider_override   = "hyperv"
    compression_level   = 1
  }
}
