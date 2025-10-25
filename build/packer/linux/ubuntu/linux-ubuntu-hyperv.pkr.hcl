packer {
  required_plugins {
    hyperv = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/hyperv"
    }
  }
}

variable "ubuntu_iso_url" {
  type    = string
  default = "https://releases.ubuntu.com/24.04.3/ubuntu-24.04.3-live-server-amd64.iso"
}

variable "ubuntu_iso_checksum" {
  type    = string
  default = "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b"
}

source "hyperv-iso" "ubuntu" {
  vm_name            = "linux-ubuntu-packer-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  output_directory   = "${path.root}/../../../../vendor/linux/hyperv-ubuntu-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  iso_url            = var.ubuntu_iso_url
  iso_checksum       = var.ubuntu_iso_checksum
  generation         = 2
  memory             = 4096
  cpus               = 8
  disk_size          = 20000
  switch_name        = "Default Switch"
  enable_secure_boot = false

  communicator = "ssh"
  ssh_username = "packer"
  ssh_password = "P@ssw0rd!"
  ssh_timeout  = "20m"

  boot_wait = "2s"

  boot_command = [
    "e<wait2>",
    "<leftShiftOn><down><down><down><end><leftShiftOff><wait2>",
    "<leftShiftOn><left><left><left><leftShiftOff><wait> autoinstall ds=nocloud-net;s=http://{{ .HTTPIP }}:{{ .HTTPPort }}/ ",
    "<wait10>",
    "<f10><wait>",
  ]

  http_directory = "${path.root}/http"

  shutdown_command = "echo 'packer' | sudo -S shutdown -P now"
}

build {
  sources = ["source.hyperv-iso.ubuntu"]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get upgrade -y"
    ]
  }
}