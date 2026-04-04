# Testing Workflows

This guide collects the platform-specific testing workflows that used to live in the main `README.md`.
Use the cross-platform matrix in the root [README](../README.md) for a quick support overview, then use the sections below for concrete commands.

## Unified VM Workflow

Most VM-backed test flows follow the same lifecycle from the repository root:

```bash
alchemy build <osname> [--type <type>] [--arch <arch>]
alchemy create <osname> [--type <type>] [--arch <arch>]
alchemy start <osname> [--type <type>] [--arch <arch>]
alchemy provision <osname> [--type <type>] [--arch <arch>] --check
alchemy provision <osname> [--type <type>] [--arch <arch>]
alchemy stop <osname> [--type <type>] [--arch <arch>]
alchemy destroy <osname> [--type <type>] [--arch <arch>]
```

- `build` creates or refreshes the reusable VM artifact.
- `create` creates the managed VM target from that artifact.
- `start` starts an existing created VM when it is stopped.
- `provision` runs the Ansible workflow against the running target.
- `stop` shuts the VM down without deleting it.
- `destroy` removes the managed VM target.

Depending on the backend, the initial boot may happen during `create` or require a small host-specific step. After a VM has been created, use `start` whenever you want to boot it again.

Use the `list` subcommands to see what your current host supports:

```bash
alchemy build list
alchemy create list
alchemy start list
alchemy provision list
alchemy stop list
alchemy destroy list
```

Use `--help` on the root command or any subcommand to inspect supported flags and usage details:

```bash
alchemy --help
alchemy create --help
alchemy start --help
alchemy provision --help
```

## System-Agnostic Docker Workflow

## Local Host Provisioning

Use the shared local wrapper when you want to apply the playbook to the current
machine instead of a managed VM:

```bash
alchemy provision local --check
alchemy provision local --proto ssh --check
alchemy provision local --proto ssh --check --yes --force-ssh-uninstall
alchemy provision local --playbook ./playbooks/bootstrap.yml
alchemy provision local -- --diff
alchemy provision local --inventory-path ./inventory/remote.yml -- --limit workstation --ask-become-pass
alchemy provision local
```

- Windows uses `inventory/localhost_windows_winrm.yml` by default.
- Windows can also use `inventory/localhost_windows_ssh.yml` with `--proto ssh`.
- macOS and Linux use `inventory/localhost.yaml`.
- Use `--playbook` to point provision runs at a different playbook file.
- Pass additional `ansible-playbook` flags after `--` when needed.
- The macOS/Linux local target is currently marked unstable until it has been
  validated end-to-end.

### Ubuntu role tests on Linux, WSL, Windows, or macOS

Use the provided Docker Compose setup to run the Ubuntu-focused Ansible playbook inside a container:

```bash
docker compose -f deployments/docker-compose/ansible/docker-compose.yml up
```

Clean up afterwards with:

```bash
docker compose -f deployments/docker-compose/ansible/docker-compose.yml down
```

## Windows Host Workflows

### Ubuntu on Windows with Hyper-V

Install host dependencies first:

```powershell
alchemy.exe install
```

Build the Ubuntu artifact:

```powershell
# server
alchemy.exe build ubuntu --type server --arch amd64
# desktop
alchemy.exe build ubuntu --type desktop --arch amd64
```

Create the VM:

```powershell
$env:VAGRANT_HYPERV_SWITCH = "Default Switch"
alchemy.exe create ubuntu --type server --arch amd64
# or desktop
alchemy.exe create ubuntu --type desktop --arch amd64
```

Provision it:

```powershell
alchemy.exe provision ubuntu --type server --arch amd64 --check
alchemy.exe provision ubuntu --type server --arch amd64
```

The command discovers the VM IP automatically and runs Ansible through the Windows/Cygwin wrapper.
Optional Ubuntu provisioning overrides can be set in `.env` using `HYPERV_UBUNTU_ANSIBLE_*`.

Related guides:

- [Ubuntu Packer README](../build/packer/linux/ubuntu/README.md)
- [Ubuntu Hyper-V deployment README](../deployments/vagrant/linux-ubuntu-hyperv/README.md)

### Windows on Windows with Docker Desktop

Use Docker Desktop with Windows containers enabled:

```bash
docker compose -f deployments/docker-compose/ansible-windows/docker-compose.yml up
```

Clean up afterwards with:

```bash
docker compose -f deployments/docker-compose/ansible-windows/docker-compose.yml down
```

More details:

- [Docker Windows Ansible README](../deployments/docker-compose/ansible-windows/README.md)

### Windows on Windows with Hyper-V

Install host dependencies first:

```powershell
alchemy.exe install
```

You will need a Windows ISO for the build. You can download one manually from Microsoft or use:

- [download_win_11.ps1](../scripts/windows/download_win_11.ps1)

Build details live here:

- [Windows Packer README](../build/packer/windows/README.md)
- [Windows Hyper-V deployment README](../deployments/vagrant/ansible-windows/README.md)

After the VM is running, provision it from the repository root:

```powershell
alchemy.exe provision windows11 --arch amd64 --check
alchemy.exe provision windows11 --arch amd64
```

Set WinRM credentials in project-root `.env` or the process environment:

```dotenv
HYPERV_WINDOWS_ANSIBLE_USER=Administrator
HYPERV_WINDOWS_ANSIBLE_PASSWORD=your-secure-password
# Optional (defaults shown):
HYPERV_WINDOWS_ANSIBLE_CONNECTION=winrm
HYPERV_WINDOWS_ANSIBLE_WINRM_TRANSPORT=basic
HYPERV_WINDOWS_ANSIBLE_PORT=5985
```

Optional shell path overrides for the Windows/Cygwin wrapper:

```powershell
$env:CYGWIN_BASH_PATH = "C:\tools\cygwin\bin\bash.exe"
# or, if you prefer setting the Cygwin terminal path:
$env:CYGWIN_TERMINAL_PATH = "C:\tools\cygwin\bin\mintty.exe"
```

Path resolution precedence for provisioning:

1. `CYGWIN_BASH_PATH`
2. `CYGWIN_TERMINAL_PATH` when `CYGWIN_BASH_PATH` is unset
3. Auto-detect `C:\tools\cygwin\bin\bash.exe`
4. Auto-detect `C:\cygwin64\bin\bash.exe`

If `CYGWIN_TERMINAL_PATH` points to `mintty.exe`, provisioning resolves it to the sibling `bash.exe`.

## macOS Host Workflows

### macOS on macOS with Tart

Use the provided script:

```bash
./scripts/macos/test-ansible-macos.sh
```

The script runs the Ansible playbook against a temporary Tart VM.
Tart project:

- https://github.com/cirruslabs/tart

By default, Dev Alchemy uses the Tart image's development credentials for Ansible access (`admin` / `admin`).
Override them in `.env` if needed:

```bash
TART_MACOS_ANSIBLE_USER=admin
TART_MACOS_ANSIBLE_PASSWORD=admin
```

`TART_MACOS_ANSIBLE_BECOME_PASSWORD` defaults to `TART_MACOS_ANSIBLE_PASSWORD` when unset.

Clean up afterwards with:

```bash
tart delete sequoia-base
```

### Windows on macOS with UTM

Install host dependencies first:

```bash
alchemy install
```

Build and create the Windows 11 VM:

```bash
# arm64 requires sudo to create a custom .iso file for automated installation.
# sudo rights are evaluated at runtime, so you can run the build command without sudo and it will ask for sudo rights only if needed.
arch=arm64 # or amd64
# sudo alchemy build windows11 --arch $arch --headless
alchemy build windows11 --arch $arch --headless
# `--headless` applies to `build`, not `create`.
alchemy create windows11 --arch $arch
```

Open UTM and start the created VM.

Set WinRM credentials in project-root `.env` or the process environment:

```dotenv
UTM_WINDOWS_ANSIBLE_USER=Administrator
UTM_WINDOWS_ANSIBLE_PASSWORD=your-secure-password
# Optional (defaults shown):
UTM_WINDOWS_ANSIBLE_CONNECTION=winrm
UTM_WINDOWS_ANSIBLE_WINRM_TRANSPORT=basic
UTM_WINDOWS_ANSIBLE_PORT=5985
```

Provision it from the repository root:

```bash
alchemy provision windows11 --arch $arch --check
alchemy provision windows11 --arch $arch
```

The wrapper discovers the VM IP automatically from the generated UTM config and `arp -a`, then runs `ansible-playbook` with an inline inventory target.
On macOS it also sets `OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES` for the Ansible process automatically.

If you need to inspect the discovered IP manually:

```bash
bash ./deployments/utm/determine-vm-ip-address.sh --arch $arch --os windows11
```

Related guide:

- [Windows Packer README](../build/packer/windows/README.md)

Newly built Windows images install a dedicated WinRM firewall rule for TCP `5985` on all network profiles, so later NIC or network-profile changes should not break reachability.
Older images may still need their network switched to `Private` or an equivalent firewall rule added manually.

### Ubuntu on macOS with UTM

Install host dependencies first:

```bash
alchemy install
```

Build and create the Ubuntu VM:

```bash
arch=arm64 # or amd64
type=desktop # or server
alchemy build ubuntu --arch $arch --type $type
alchemy create ubuntu --arch $arch --type $type
```

Open UTM and start the created VM.

Set Ubuntu SSH credentials in project-root `.env` or the process environment:

```dotenv
UTM_UBUNTU_ANSIBLE_USER=packer
UTM_UBUNTU_ANSIBLE_PASSWORD=P@ssw0rd!
UTM_UBUNTU_ANSIBLE_BECOME_PASSWORD=P@ssw0rd!
# Optional (defaults shown):
UTM_UBUNTU_ANSIBLE_CONNECTION=ssh
UTM_UBUNTU_ANSIBLE_SSH_COMMON_ARGS=-o StrictHostKeyChecking=no -o ServerAliveInterval=10 -o ServerAliveCountMax=3 -o ControlMaster=no -o ControlPersist=no
UTM_UBUNTU_ANSIBLE_SSH_TIMEOUT=120
UTM_UBUNTU_ANSIBLE_SSH_RETRIES=3
```

Provision it from the repository root:

```bash
alchemy provision ubuntu --type $type --arch $arch --check
alchemy provision ubuntu --type $type --arch $arch
```

The wrapper discovers the VM IP automatically from the generated UTM config and `arp -a`, then runs `ansible-playbook` with an inline inventory target.
On macOS it also sets `OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES` for the Ansible process automatically.

If you need to inspect the discovered IP manually:

```bash
bash ./deployments/utm/determine-vm-ip-address.sh --arch $arch --os "ubuntu-$type"
```
