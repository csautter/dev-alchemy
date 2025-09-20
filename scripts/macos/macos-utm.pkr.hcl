source "null" "utm_macos" {
  communicator = "ssh"
  ssh_username = "packer"
  ssh_password = "packer"
  ssh_timeout  = "10m"
  ssh_host = "localhost"
}

build {
  name    = "macos-utm-build"
  sources = ["source.null.utm_macos"]

  provisioner "shell-local" {
    command = "./scripts/macos/create_utm_vm.sh"
  }

  provisioner "shell" {
    inline = [
      "echo 'Provisioning macOS VM after install...'",
      "brew install ansible",
      "echo 'Done provisioning.'"
    ]
  }
}
