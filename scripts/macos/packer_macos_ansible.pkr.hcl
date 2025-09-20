packer {
  required_plugins {
    qemu = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/qemu"
    }
    ansible = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/ansible"
    }
  }
}

source "qemu" "macos-qemu" {
  iso_url          = "./vendor/macos/macos_installer_Sequoia.iso"
  iso_checksum     = "none"
  output_directory = "output-macos-qemu"
  disk_size        = 64000
  format           = "qcow2"
  headless         = false
  accelerator      = "hvf"
  ssh_username     = "admin"
  ssh_password     = "password123"
  ssh_wait_timeout = "60m"
  qemuargs = [
    ["-cpu", "host"],
    ["-smp", "4"],
    ["-m", "4096"],
    ["-machine", "q35"],
    ["-display", "cocoa"],
    ["-cpu", "Penryn"],
    ["-boot", "order=d"],
    ["-drive", "if=virtio,file=./output-macos/packer.qcow2,format=qcow2"],
    ["-drive", "if=ide,index=0,media=cdrom,file=OpenCore.iso"],
    ["-drive", "if=ide,index=2,media=cdrom,file=./vendor/macos/macos_installer_Sequoia.iso"]
  ]
}

build {
  sources = ["source.qemu.macos-qemu"]

  provisioner "ansible" {
    playbook_file   = "playbooks/setup.yml"
    extra_arguments = [
      "--connection=ssh",
      "--ssh-extra-args='-o StrictHostKeyChecking=no'",
      "-i inventory/localhost.yml",
      "--check"
    ]
  }
}