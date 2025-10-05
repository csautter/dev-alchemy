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

The build process will generate a Windows image in Vagrant box format as defined in [windows.pkr.hcl](windows.pkr.hcl).

## Build Windows on macOS Hosts
This directory also contains a Packer template for building Windows images on macOS hosts using QEMU.

```bash
packer init build/packer/windows/windows11-x86-on-macos.pkr.hcl
packer build -var "iso_url=../../../vendor/windows/Win11_25H2_English_x64.iso" build/packer/windows/windows11-x86-on-macos.pkr.hcl