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
  default = "https://mirrors.kernel.org/linuxmint/stable/22.2/linuxmint-22.2-cinnamon-64bit.iso"
}

variable "iso_checksum" {
  type    = string
  default = "sha256:759c9b5a2ad26eb9844b24f7da1696c705ff5fe07924a749f385f435176c2306"
}

source "hyperv-iso" "linuxmint" {
  vm_name          = "linux-mint-packer-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"
  output_directory = "${path.root}/../../../../vendor/linux/hyperv-mint-output-${formatdate("YYYY-MM-DD-hh-mm", timestamp())}"

  iso_url          = var.iso_url
  iso_checksum     = var.iso_checksum
  generation       = 2
  ssh_username     = "packer"
  ssh_password     = "P@ssw0rd!"
  ssh_timeout      = "60m"
  shutdown_command = "echo 'packer' | sudo -S shutdown -P now"
  shutdown_timeout = "1m"
  disk_size        = 20000
  memory           = 8192
  cpus             = 8

  switch_name = "Default Switch"

  boot_wait = "2s"

  boot_command = [
    "<enter><wait40>",                                           # initial boot screen
    "<leftCtrlOn><leftAltOn>t<leftCtrlOff><leftAltOff><wait10>", # open terminal
    "wget -O /tmp/preseed.cfg http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg<enter><wait>",
    "sudo debconf-set-selections /tmp/preseed.cfg<enter><wait>",
    "ubiquity --automatic gtk_ui<enter><wait40>",
  ]
  http_directory = "${path.root}/http"
}

build {
  sources = ["source.hyperv-iso.linuxmint"]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get upgrade -y",
      "sudo apt-get install -y build-essential"
    ]
  }

  post-processor "vagrant" {
    output              = "${path.root}/../../../../vendor/windows/linux-mint-hyperv.box"
    keep_input_artifact = false
    provider_override   = "hyperv"
    compression_level   = 1
  }
}