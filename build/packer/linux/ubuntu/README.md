# Ubuntu Packer Templates

This directory contains Packer templates used by the Go wrapper to build Ubuntu images.

## Build Ubuntu on Windows Hosts (Hyper-V)

Use the wrapper from repository root:

```powershell
go run cmd/main.go install

# build ubuntu server (Hyper-V)
go run cmd/main.go build ubuntu --type server --arch amd64
# build ubuntu desktop (Hyper-V)
go run cmd/main.go build ubuntu --type desktop --arch amd64
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

Use the wrapper from repository root:

```bash
# amd64 or arm64
arch=amd64
go run cmd/main.go install
go run cmd/main.go build ubuntu --type server --arch $arch
go run cmd/main.go build ubuntu --type desktop --arch $arch
```

Manual script usage:

```bash
export DEV_ALCHEMY_APP_DATA_DIR="${DEV_ALCHEMY_APP_DATA_DIR:-$HOME/Library/Application Support/dev-alchemy}"
build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh --project-root "$PWD" --arch amd64 --ubuntu-type server
build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh --project-root "$PWD" --arch amd64 --ubuntu-type desktop
```
