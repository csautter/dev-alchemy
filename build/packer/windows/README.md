# Windows Packer Template

## Build Windows on Windows Hosts

This directory contains the Packer templates for building Windows 11 images on Windows and macOS hosts.

### Prerequisites

- [Packer](https://www.packer.io/downloads) installed
- Windows host or compatible environment
- For repository-managed host dependencies, run `go run cmd/main.go install` from repo root on macOS or Windows before building.

### Usage

For manual builds on Windows, use the current Hyper-V or VirtualBox templates and point them at the managed Dev Alchemy cache.

```powershell
$AppDataDir = if ($env:DEV_ALCHEMY_APP_DATA_DIR) { $env:DEV_ALCHEMY_APP_DATA_DIR } else { Join-Path $env:LOCALAPPDATA "dev-alchemy" }
$CacheDir = Join-Path $AppDataDir "cache"
$env:DEV_ALCHEMY_CACHE_DIR = $CacheDir
$env:DEV_ALCHEMY_PACKER_CACHE_DIR = Join-Path $AppDataDir "packer_cache"

# Find newest ISO file in the managed cache
$isoPath = Get-ChildItem -Path (Join-Path $CacheDir "windows11\iso") -Filter "Win11_*.iso" | Sort-Object LastWriteTime -Descending | Select-Object -First 1 | Select-Object -ExpandProperty FullName
Write-Host "Using ISO: $isoPath"
```

To build the Windows image, run:

```powershell
# Hyper-V
packer init build/packer/windows/windows11-on-windows-hyperv.pkr.hcl
packer build -var "cache_dir=$CacheDir" -var "iso_url=$isoPath" build/packer/windows/windows11-on-windows-hyperv.pkr.hcl

# VirtualBox
packer init build/packer/windows/windows11-on-windows-virtualbox.pkr.hcl
packer build -var "cache_dir=$CacheDir" -var "iso_url=$isoPath" build/packer/windows/windows11-on-windows-virtualbox.pkr.hcl
```

You can reduce build time by disabling compression in the `post-processor "vagrant"` block of the relevant template and setting `compression_level = 0`.
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
export DEV_ALCHEMY_APP_DATA_DIR="${DEV_ALCHEMY_APP_DATA_DIR:-$HOME/Library/Application Support/dev-alchemy}"
go run cmd/main.go install
go run cmd/main.go build windows11 --arch $arch
go run cmd/main.go create windows11 --arch $arch
```

Start the VM in UTM and provision it from repository root with the unified wrapper:

```bash
go run cmd/main.go provision windows11 --arch $arch --check
go run cmd/main.go provision windows11 --arch $arch
```

Set the required WinRM credentials in project-root `.env` or process environment before provisioning:

```dotenv
UTM_WINDOWS_ANSIBLE_USER=Administrator
UTM_WINDOWS_ANSIBLE_PASSWORD=your-secure-password
```

### Security Note

Current Windows templates keep WinRM provisioning reachable even when Windows reclassifies the NIC as `Public`. They do that by enabling WinRM Basic over HTTP, allowing unencrypted WSMan traffic, and creating an inbound firewall rule for TCP `5985` with `Profile Any` and `RemoteAddress Any`.

That choice broadens the reachable attack surface compared with private-profile-only access. During unattended setup, the built-in `Administrator` credential is also configured in the answer file, so these images should only be built and booted on trusted, isolated networks.

A safer provisioning approach is planned for future releases. Until then, treat the current WinRM path as a compatibility tradeoff rather than a hardened default.
