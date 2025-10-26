# Ubuntu Packer Template

## Build Ubuntu on Windows Hosts

This directory contains a Packer template for building Ubuntu images.

### Prerequisites

- [Packer](https://www.packer.io/downloads) installed
- Windows host or compatible environment

### Usage

To build the Ubuntu image, run:

```powershell
# build ubuntu server
packer build -var "desktop_version=false" build/packer/linux/ubuntu/linux-ubuntu-hyperv.pkr.hcl
# build ubuntu desktop
packer build -var "desktop_version=true" build/packer/linux/ubuntu/linux-ubuntu-hyperv.pkr.hcl
```

You can reduce build time by disabling compression in the Vagrant post-processor. Edit the `compression_level` in the `post-processor "vagrant"` block of [windows.pkr.hcl](windows.pkr.hcl) and set it to `0` for no compression.
Default for packer is `6`.
[Compression Level Reference](https://developer.hashicorp.com/packer/docs/post-processors/compress#compression_level)

### Output

The build process will generate a Ubuntu image in Vagrant box format as defined in [linux-ubuntu-hyperv.pkr.hcl](linux-ubuntu-hyperv.pkr.hcl).
