# Windows Packer Template

This directory contains a Packer template for building Windows images.

## Prerequisites

- [Packer](https://www.packer.io/downloads) installed
- Windows host or compatible environment

## Usage

To build the Windows image, run:

```powershell
packer build windows.pkr.hcl
```

## Output

The build process will generate a Windows image in Vagrant box format as defined in [windows.pkr.hcl](windows.pkr.hcl).
