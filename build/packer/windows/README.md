# Windows Packer Template

## Build Windows on Windows Hosts

This directory contains a Packer template for building Windows images.

### Prerequisites

- [Packer](https://www.packer.io/downloads) installed
- Windows host or compatible environment

### Usage

Set the iso_url variable in [windows.pkr.hcl](windows.pkr.hcl) to point to your Windows ISO file.

```powershell
# Example for Windows 11 ISO
$isoPath = "C:\path\to\your\Win11_*.iso"

# Find newest iso file in vendor/windows directory
$isoPath = Get-ChildItem -Path ".\vendor\windows" -Filter "Win11_*.iso" | Sort-Object LastWriteTime -Descending | Select-Object -First 1 | Select-Object -ExpandProperty FullName
Write-Host "Using ISO: $isoPath"
```

To build the Windows image, run:

```powershell
# with default iso_url from windows.pkr.hcl
packer build build/packer/windows/windows.pkr.hcl
# or override iso_url
packer build -var "iso_url=$isoPath" build/packer/windows/windows.pkr.hcl
```

You can reduce build time by disabling compression in the Vagrant post-processor. Edit the `compression_level` in the `post-processor "vagrant"` block of [windows.pkr.hcl](windows.pkr.hcl) and set it to `0` for no compression.
Default for packer is `6`.
[Compression Level Reference](https://developer.hashicorp.com/packer/docs/post-processors/compress#compression_level)

### Output

## Build Windows on macOS Hosts

The windows VM build process is fully automated and includes installation of:

- Download Windows iso
- Unattended Windows installation
- Qemu Guest Additions
- WinRM enabled
- SSH server installed
- UTM VM

### run build and create

After running following commands, the Windows 11 VM will be available in UTM.
Every command may take a while to finish. If something goes wrong, please check the logs and retry.
The process is idempotent, so you can re-run commands without issues.

```bash
arch=arm64 # or amd64
go run cmd/main.go build windows11 --arch $arch
go run cmd/main.go create windows11 --arch $arch
```
