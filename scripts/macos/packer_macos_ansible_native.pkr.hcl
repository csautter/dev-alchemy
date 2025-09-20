packer {
  required_plugins {
    mac = {
      version = ">= 0.1.0"
      source  = "github.com/macstadium/mac"
    }
  }
}

source "mac" "ventura" {
  iso_path       = "/Applications/Install macOS Ventura.app"
  boot_command   = []
  ssh_username   = "packer"
  ssh_password   = "packer"
  disk_size      = "40G"
  memory         = "4096"
  cpu_count      = 4
}

build {
  sources = ["source.mac.ventura"]

  provisioner "shell" {
    inline = ["echo Hello from inside the macOS VM!"]
  }
}
