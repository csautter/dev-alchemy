packer {
  required_plugins {
    qemu = {
      version = ">= 1.1.4"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

variable "ubuntu_version" {
  type    = string
  default = "24.04.3"
}

variable "host_arch" {
  type        = string
  description = "Normalized host architecture: amd64 or arm64."
  validation {
    condition     = var.host_arch == "amd64" || var.host_arch == "arm64"
    error_message = "The variable host_arch must be either 'amd64' or 'arm64'."
  }
}

variable "arch" {
  type        = string
  default     = "amd64"
  description = "Target architecture: amd64 or arm64."
  validation {
    condition     = var.arch == "amd64" || var.arch == "arm64"
    error_message = "The variable arch must be either 'amd64' or 'arm64'."
  }
}

variable "headless" {
  type    = bool
  default = false
}

variable "vnc_port" {
  type    = number
  default = 5901
}

variable "iso_url" {
  type = string
}

variable "use_hardware_acceleration" {
  type        = bool
  default     = true
  description = "Whether to use KVM acceleration when the host can support it."
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
  description = "Memory in MB to allocate to the VM"
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

variable "build_output_dir" {
  type        = string
  default     = ""
  description = "Optional short-lived Packer output directory."
}

locals {
  iso_url             = var.iso_url
  ubuntu_iso_checksum = var.arch == "amd64" ? "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b" : "none"
  cache_directory     = var.cache_dir
  host_same_arch      = var.host_arch == var.arch

  amd64_can_use_native_acceleration = local.host_same_arch && var.use_hardware_acceleration
  amd64_accel                       = local.amd64_can_use_native_acceleration ? "kvm" : "tcg,thread=multi,tb-size=1024"
  amd64_cpu_model                   = local.amd64_can_use_native_acceleration ? "host" : "Skylake-Client"

  arm64_can_use_native_acceleration = local.host_same_arch && var.use_hardware_acceleration
  arm64_software_accel              = "tcg,thread=multi,tb-size=1024"
  arm64_fallback_cpu_model          = "max,sve=off,sme=off,pauth-impdef=on"

  arm64_accel     = local.arm64_can_use_native_acceleration ? "kvm" : local.arm64_software_accel
  arm64_cpu_model = local.arm64_can_use_native_acceleration ? "host" : local.arm64_fallback_cpu_model

  boot_command = {
    "amd64" = [
      "e<wait2>",
      "<leftShiftOn><down><down><down><end><leftShiftOff><wait2>",
      "<leftShiftOn><left><left><left><leftShiftOff><wait> autoinstall ds=nocloud ",
      "<wait2>",
      "<f10><wait>",
    ]
    "arm64" = [
      "e<wait2>",
      "<down><down><down><end><wait2>",
      "${local.left_list}<wait> autoinstall ds=nocloud ",
      "<wait2>",
      "<f10><wait>",
    ]
  }

  qemu_args = {
    "amd64" = [
      ["-machine", "q35,vmport=off,i8042=off,hpet=off"],
      ["-accel", local.amd64_accel],
      ["-smp", "cpus=${var.cpus},cores=${var.cpus},sockets=1,threads=1"],
      ["-global", "PIIX4_PM.disable_s3=1"],
      ["-global", "ICH-LPC.disable_s3=1"],
      ["-device", "qemu-xhci"],
      ["-device", "usb-kbd"],
      ["-device", "usb-tablet"],
      ["-device", "usb-mouse"],
    ]
    "arm64" = [
      ["-accel", local.arm64_accel],
      ["-machine", "virt,highmem=on"],
      ["-cpu", local.arm64_cpu_model],
      ["-bios", "${local.cache_directory}/qemu-uefi/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"],
      ["-device", "ramfb"],
      ["-smp", "cpus=${var.cpus},cores=${var.cpus},sockets=1,threads=1"],
      ["-global", "PIIX4_PM.disable_s3=1"],
      ["-global", "ICH-LPC.disable_s3=1"],
      ["-device", "qemu-xhci"],
      ["-device", "usb-kbd"],
      ["-device", "usb-tablet"],
      ["-device", "usb-mouse"],
      ["-device", "virtio-blk-pci,drive=cdrom,bootindex=1"],
      ["-drive", "if=none,id=cdrom,media=cdrom,file=${local.iso_url},readonly=true"],
      ["-device", "virtio-blk-pci,drive=disk,serial=deadbeef,bootindex=0"],
      ["-drive", "if=none,media=disk,id=disk,format=qcow2,file.filename=${local.cache_directory}/ubuntu/qemu-ubuntu-${var.ubuntu_type}-packer-${var.arch}.qcow2,discard=unmap,detect-zeroes=unmap"],
      ["-drive", "if=none,id=cidata,format=raw,file=${path.root}/cloud-init/qemu-${var.ubuntu_type}/cidata.iso,readonly=true"],
      ["-device", "virtio-blk-pci,drive=cidata"],
      ["-boot", "order=d,menu=on"],
    ]
  }

  left_list        = join("", [for i in range(0, 16) : "<left>"])
  output_directory = var.build_output_dir != "" ? var.build_output_dir : "${local.cache_directory}/ubuntu/qemu-out-ubuntu-${var.ubuntu_type}-${var.arch}"

  # Packages are installed via Packer provisioners (instead of cloud-init) to
  # improve reliability under cross-architecture TCG emulation where the
  # autoinstall phase is extremely slow and has no retry mechanism.
  base_packages = compact(concat(
    ["openssh-server", "linux-virtual", "linux-tools-virtual", "linux-cloud-tools-common", "net-tools", "qemu-guest-agent", "spice-vdagent"],
    var.ubuntu_type == "server" ? ["linux-tools-generic"] : []
  ))
  desktop_packages = compact(concat(
    ["ubuntu-desktop-minimal", "gdm3", "network-manager"],
    var.arch == "amd64" ? ["xserver-xorg-video-qxl"] : []
  ))
}

source "qemu" "ubuntu" {
  qemu_binary      = var.arch == "amd64" ? "qemu-system-x86_64" : "qemu-system-aarch64"
  vm_name          = "linux-ubuntu-${var.ubuntu_type}-packer-${var.arch}"
  headless         = var.headless
  output_directory = local.output_directory
  iso_url          = local.iso_url
  iso_checksum     = local.ubuntu_iso_checksum
  memory           = var.memory
  cpu_model        = var.arch == "amd64" ? local.amd64_cpu_model : local.arm64_cpu_model
  disk_size        = "64G"
  disk_interface   = "ide"
  format           = "qcow2"
  display          = "none"
  net_device       = var.arch == "amd64" ? "e1000" : "virtio-net-pci"

  cd_label = "cidata"
  cd_files = [
    "${path.root}/cloud-init/qemu-${var.ubuntu_type}/meta-data",
    "${path.root}/cloud-init/qemu-${var.ubuntu_type}/user-data"
  ]

  vnc_bind_address = "127.0.0.1"
  vnc_port_min     = var.vnc_port
  vnc_port_max     = var.vnc_port
  vnc_use_password = true
  vnc_password     = "packer"

  communicator = "ssh"
  ssh_username = "packer"
  ssh_password = "P@ssw0rd!"
  ssh_timeout  = "4h"

  boot_wait        = var.arch == "amd64" ? "2s" : "10s"
  boot_command     = local.boot_command[var.arch]
  shutdown_command = "echo 'P@ssw0rd!' | sudo -S shutdown -P now"
  qemuargs         = local.qemu_args[var.arch]
}

build {
  sources = ["source.qemu.ubuntu"]

  provisioner "shell" {
    environment_vars = ["SUDO_ASKPASS=/tmp/askpass.sh"]
    inline = [
      "printf '#!/bin/sh\necho '\"'\"'P@ssw0rd!'\"'\"'\n' > /tmp/askpass.sh && chmod +x /tmp/askpass.sh",
      "echo 'Waiting for cloud-init to finish...'",
      "sudo -A cloud-init status --wait || true",
    ]
    pause_before = "10s"
    timeout      = "30m"
  }

  provisioner "shell" {
    environment_vars = ["DEBIAN_FRONTEND=noninteractive", "SUDO_ASKPASS=/tmp/askpass.sh"]
    inline = [
      "echo 'Updating package lists...'",
      "sudo -A apt-get update -q",
      "echo 'Installing base packages...'",
      "sudo -A apt-get install -y ${join(" ", local.base_packages)}",
    ]
    max_retries  = 3
    pause_before = "10s"
    timeout      = "60m"
  }

  provisioner "shell" {
    environment_vars = ["DEBIAN_FRONTEND=noninteractive", "SUDO_ASKPASS=/tmp/askpass.sh"]
    inline = var.ubuntu_type == "desktop" ? [
      "echo 'Installing desktop environment and graphics integration packages without recommended packages...'",
      "sudo -A apt-get install -y --no-install-recommends ${join(" ", local.desktop_packages)}",
    ] : ["echo 'Server build - skipping desktop packages.'"]
    max_retries  = 2
    pause_before = "10s"
    timeout      = "120m"
  }

  provisioner "shell" {
    environment_vars = ["SUDO_ASKPASS=/tmp/askpass.sh"]
    inline = var.ubuntu_type == "desktop" ? [
      "echo 'Configuring desktop netplan to use NetworkManager on future boots...'",
      "printf 'network:\\n  version: 2\\n  renderer: NetworkManager\\n' > /tmp/90-dev-alchemy-networkmanager.yaml",
      "sudo -A install -m 600 /tmp/90-dev-alchemy-networkmanager.yaml /etc/netplan/90-dev-alchemy-networkmanager.yaml",
      "sudo -A systemctl enable NetworkManager.service",
    ] : ["echo 'Server build - keeping default netplan renderer.'"]
    pause_before = "5s"
    timeout      = "10m"
  }

  post-processor "shell-local" {
    inline = var.arch == "amd64" ? [
      "echo 'Exporting QCOW2 image...'",
      "mkdir -p \"${local.cache_directory}/ubuntu\"",
      "cp \"${local.output_directory}\"/linux-ubuntu-${var.ubuntu_type}-packer-* \"${local.cache_directory}/ubuntu/qemu-ubuntu-${var.ubuntu_type}-packer-${var.arch}.qcow2\"",
      "echo 'Export completed.'"
      ] : [
      "echo 'No export needed for arm64 architecture.'"
    ]
  }
}
