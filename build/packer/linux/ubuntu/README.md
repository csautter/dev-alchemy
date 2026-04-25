# Ubuntu Packer Templates

This directory contains Packer templates used by the Dev Alchemy CLI to build Ubuntu images.

## Build Ubuntu on Windows Hosts (Hyper-V)

Use the CLI from repository root:

```powershell
alchemy.exe install

# build ubuntu server (Hyper-V)
alchemy.exe build ubuntu --type server --arch amd64
# build ubuntu desktop (Hyper-V)
alchemy.exe build ubuntu --type desktop --arch amd64
```

The Hyper-V template is [linux-ubuntu-hyperv.pkr.hcl](linux-ubuntu-hyperv.pkr.hcl).
Both server and desktop builds use the Ubuntu **live-server** ISO for unattended installation.
Hyper-V cloud-init input is split by type:
- `cloud-init/hyperv-server/meta-data` + `cloud-init/hyperv-server/user-data`
- `cloud-init/hyperv-desktop/meta-data` + `cloud-init/hyperv-desktop/user-data`
Both Hyper-V variants use cloud-init apt offline mode (`fallback: offline-install`, `geoip: false`) like the QEMU variants.

Manual Packer usage:

```powershell
$AppDataDir = if ($env:DEV_ALCHEMY_APP_DATA_DIR) { $env:DEV_ALCHEMY_APP_DATA_DIR } else { Join-Path $env:LOCALAPPDATA "dev-alchemy" }
$CacheDir = Join-Path $AppDataDir "cache"
$env:DEV_ALCHEMY_CACHE_DIR = $CacheDir
$env:DEV_ALCHEMY_PACKER_CACHE_DIR = Join-Path $AppDataDir "packer_cache"
$isoPath = Join-Path $CacheDir "linux\ubuntu-24.04.3-live-server-amd64.iso"

packer init build/packer/linux/ubuntu/linux-ubuntu-hyperv.pkr.hcl

# server
packer build -var "cache_dir=$CacheDir" -var "ubuntu_type=server" -var "iso_url=$isoPath" build/packer/linux/ubuntu/linux-ubuntu-hyperv.pkr.hcl

# desktop
packer build -var "cache_dir=$CacheDir" -var "ubuntu_type=desktop" -var "iso_url=$isoPath" build/packer/linux/ubuntu/linux-ubuntu-hyperv.pkr.hcl
```

Output boxes:

- `%LOCALAPPDATA%\dev-alchemy\cache\ubuntu\hyperv-ubuntu-server-amd64.box`
- `%LOCALAPPDATA%\dev-alchemy\cache\ubuntu\hyperv-ubuntu-desktop-amd64.box`

## Build Ubuntu on macOS Hosts (UTM/QEMU)

Use the CLI from repository root:

```bash
# amd64 or arm64
arch=amd64
alchemy install # --with-go also bootstraps the Go toolchain on macOS
alchemy build list # shows whether the Ubuntu build artifacts are already present in the local cache
alchemy build ubuntu --type server --arch $arch
alchemy build ubuntu --type desktop --arch $arch
```

Manual script usage:

```bash
export DEV_ALCHEMY_APP_DATA_DIR="${DEV_ALCHEMY_APP_DATA_DIR:-$HOME/Library/Application Support/dev-alchemy}"
build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh --project-root "$PWD" --arch amd64 --ubuntu-type server
build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh --project-root "$PWD" --arch amd64 --ubuntu-type desktop
```

## Build Ubuntu on Linux Hosts (QEMU)

Install host dependencies first:

```bash
alchemy install
```

This runs
[scripts/linux/dev-alchemy-install-dependencies.sh](../../../../scripts/linux/dev-alchemy-install-dependencies.sh)
to install the Ubuntu/Debian packages needed by the Linux QEMU workflow.

Use the CLI from repository root:

```bash
# amd64 or arm64
arch=amd64
alchemy build ubuntu --type server --arch "$arch"
alchemy build ubuntu --type desktop --arch "$arch"
alchemy create ubuntu --type server --arch "$arch"
alchemy start ubuntu --type server --arch "$arch"
```

Manual script usage:

```bash
export DEV_ALCHEMY_APP_DATA_DIR="${DEV_ALCHEMY_APP_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/dev-alchemy}"
build/packer/linux/ubuntu/linux-ubuntu-on-linux.sh --project-root "$PWD" --arch amd64 --ubuntu-type server
build/packer/linux/ubuntu/linux-ubuntu-on-linux.sh --project-root "$PWD" --arch amd64 --ubuntu-type desktop
```

The Linux `create`/`start`/`stop`/`destroy` flow uses libvirt so the VM appears
in `virt-manager`.

Linux libvirt runtime is intentionally native-architecture only:
- `amd64` guests must run on `amd64` Linux hosts
- `arm64` guests must run on `arm64` Linux hosts

Ubuntu QEMU images include `qemu-guest-agent` and `spice-vdagent`, and the
libvirt domain enables the SPICE agent channel so `virt-manager` can provide
better clipboard, pointer, and dynamic display resize integration for desktop
guests.

For `amd64` Ubuntu desktop guests, the image also includes
`xserver-xorg-video-qxl` and the Linux libvirt runtime uses a `qxl` video
device to improve SPICE auto-resize behavior in `virt-manager`.

- Default libvirt connection: `qemu:///system`
- Default managed disk directory for that connection: `/var/tmp/dev-alchemy/libvirt/images`
- Optional override for rootless libvirt user-session VMs: `DEV_ALCHEMY_LIBVIRT_URI=qemu:///session`
- Optional managed disk directory override: `DEV_ALCHEMY_LIBVIRT_IMAGE_DIR=/path/to/images`
