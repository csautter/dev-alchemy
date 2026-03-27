# 🧪 devalchemy

**devalchemy** is a cross-platform development environment automation toolkit powered
by [Ansible](https://www.ansible.com/). It turns fresh and also existing machines into fully-configured dev setups — whether you're on **macOS**, **Linux**, or **Windows**.

> _"Transform your system into a dev powerhouse — with a touch of automation magic."_

## ✨ Features

- ✅ Unified setup for macOS, Linux, and Windows
- 📦 Install development tools, CLIs, languages, and more
- ⚙️ Easily extensible Ansible roles and playbooks
- 💻 Consistent dev experience across all platforms
- 🔒 Minimal privileges needed (no full root where not required)
- 🐳 Automated cross-platform testing via Docker and VMs

---

## Evolutionary Background

**devalchemy** was born out of the need for a streamlined, consistent development environment across different platforms. As developers ourselves, we understand the pain points of setting up and maintaining development environments. With **devalchemy**, we aim to simplify this process, allowing developers to focus on what they do best: writing code.

Painpoints addressed:

- Inconsistent setups across different OSes
- Time-consuming manual installations
- Faster onboarding of new team members
- Difficulty in maintaining and updating dev environments
- Failing development setups are not reproducible.
- Over-reliance on OS-specific scripts
- Security concerns with elevated privileges

## Base Concepts

The core idea of **devalchemy** is to use Ansible playbooks and roles to define and automate the setup of development environments. This includes installing essential tools, configuring settings, and managing dependencies.<br>
Every role is platform independent and can be applied to macOS, Linux, and Windows. The playbooks are designed to be modular, allowing users to pick and choose which components they want to install.<br>
The setup is idempotent, meaning you can run the playbooks multiple times without causing issues or duplications. This ensures that your development environment remains consistent and up-to-date.<br>
Despite the common use of Ansible in server environments where changes are **pushed** from a central location, **devalchemy** is designed for local **pull** based execution on individual machines. This approach allows developers to maintain control over their own environments while still benefiting from automation. Every ansible run can be simulated with `--check` to see what changes would be applied.

---

## 🚀 Getting Started

### 1. Clone the repo

```bash
git clone https://github.com/csautter/dev-alchemy.git
cd dev-alchemy
```

### 2. Install Host Dependencies

Use the unified CLI command from the repository root.

#### macOS

```bash
go run cmd/main.go install
```

This runs [scripts/macos/dev-alchemy-install-dependencies.sh](./scripts/macos/dev-alchemy-install-dependencies.sh).

#### Ubuntu / Debian

The `install` command is currently intended for macOS and Windows hosts. On Linux, install Ansible manually:

```bash
sudo apt update && sudo apt install ansible
```

#### Discover supported targets

Use the `list` subcommands to see what the current host can build, create, or provision before running a longer workflow:

```bash
go run cmd/main.go build list
go run cmd/main.go create list
go run cmd/main.go provision list
```

#### Windows

Run the command in an elevated PowerShell session (Run as Administrator):

```powershell
go run cmd/main.go install
```

This runs [scripts/windows/dev-alchemy-self-setup.ps1](./scripts/windows/dev-alchemy-self-setup.ps1).

To force a VM rebuild even when the cached build artifact already exists, use:

```bash
go run cmd/main.go build windows11 --arch amd64 --no-cache
```

#### Managed application data

VM build and deployment state is now stored outside the repository in an OS-appropriate app-data directory:

- macOS: `~/Library/Application Support/dev-alchemy`
- Windows: `%LOCALAPPDATA%\dev-alchemy`
- Linux: `${XDG_DATA_HOME:-~/.local/share}/dev-alchemy`

Under that root, Dev Alchemy manages:

- `cache/` for downloaded files and build artifacts
- `.vagrant/` for isolated Vagrant state
- `packer_cache/` for Packer plugin/download cache

You can override the default location by setting `DEV_ALCHEMY_APP_DATA_DIR`. Dev Alchemy also exports `DEV_ALCHEMY_CACHE_DIR`, `DEV_ALCHEMY_VAGRANT_DIR`, and `DEV_ALCHEMY_PACKER_CACHE_DIR` for helper scripts and manual workflows.



##### Enable ansible remote access on Windows

On Windows you need to enable remote access for ansible to work. Also for local runs ansible needs to connect to the local host via SSH or WinRM. Don't activate the options if you don't need them.

> ⚠️ **Security note:** Enabling unencrypted WinRM and basic auth can expose your system to security risks. Use these settings only in trusted environments or for testing purposes. For production environments, consider using encrypted connections and more secure authentication methods. Keep your firewall settings in mind and only allow connections from trusted networks.

> ℹ️ For Windows there are two options to connect to the target host: via SSH or via WinRM.

The WinRM option is more native to Windows, but requires some additional setup on the target host. It might also be blocked by company policies.

```powershell
# Enable WinRM
Set-Item -Path WSMan:\localhost\Service\AllowUnencrypted -Value $true; \
Set-Item -Path WSMan:\localhost\Service\Auth\Basic -Value $true; \
Enable-PSRemoting -Force
```

The SSH option requires an SSH server to be installed on the target host, but is easier to set up.

```powershell
# Enable SSH Server
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0; \
Start-Service sshd; Set-Service -Name sshd -StartupType 'Automatic';
```

In both cases you might need to adjust the firewall settings to allow incoming connections. Also make sure to use a user with admin privileges. Create a new user if needed.

```powershell
# ⚠️ OPTIONAL - use your existing user if possible
# Create new user
# And of course set a secure password!
net user ansible 'Secret123!@#' /add; \
net localgroup Administrators ansible /add
```

---

### 3. Run the Playbook

#### Run the Playbook on localhost

Dry run to check for issues:

```bash
ansible-playbook playbooks/setup.yml -i inventory/localhost.yaml --check
```

```bash
ansible-playbook playbooks/setup.yml -i inventory/localhost.yaml
```

#### Run the Playbook on a remote host or in a VM

Dry run to check for issues:

```bash
HOST="192.168.179.21"
# write to inventory file
cat <<EOF > inventory/remote.yml
all:
  hosts:
    $HOST:
      ansible_user: admin
EOF
ansible-playbook playbooks/setup.yml -i inventory/remote.yml -l "$HOST" --ask-pass --ask-become-pass --check
```

Apply the playbook:

```bash
ansible-playbook playbooks/setup.yml -i inventory/remote.yml -l "$HOST" --ask-pass --ask-become-pass
```

You can customize the inventory or pass variables via CLI.

#### Run the Playbook on Windows

Apply the playbook via WinRM:

```powershell
$DevAlchemyPath = "C:\path\to\dev-alchemy"
C:\\cygwin64\\bin\\bash.exe -l -c "cd $DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_winrm.yml -l windows_host"
```

Apply the playbook via SSH:

```powershell
$DevAlchemyPath = "C:\path\to\dev-alchemy"
C:\\cygwin64\\bin\\bash.exe -l -c "cd $DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_ssh.yml -l windows_host --ask-pass -vvv"
```

## 🧩 Structure

```
devalchemy/
├── roles/
│   ├── role/
│   ├── role2/
│   └── role3/
├── inventory/
│   └── localhost.yaml
├── playbooks/
│   └── setup.yml
└── README.md
```

---

## 🛠️ Customization

- Add or tweak roles in `roles/`

- Use tags to run specific parts:

  ```bash
  ansible-playbook playbooks/setup.yml --tags "dotfiles,python"
  ```

- Pass variables:

  ```bash
  ansible-playbook playbooks/setup.yml -e "install_docker=true"
  ```

---

## Testing

### 🧪 Cross-Platform Testing Matrix

| Host OS     |                                        Test Linux                                         |              Test macOS              |                                                       Test Windows                                                       |
| ----------- | :---------------------------------------------------------------------------------------: | :----------------------------------: | :----------------------------------------------------------------------------------------------------------------------: |
| **macOS**   | Docker<br><sub>✅ Implemented</sub><br>\_\_\_<br>UTM Qemu VM<br><sub>✅ Implemented</sub> | Tart VM<br><sub>✅ Implemented</sub> |                                         UTM Qemu VM<br><sub>✅ Implemented</sub>                                         |
| **Linux**   |                            Docker<br><sub>✅ Implemented</sub>                            |                 ---                  |                                  VM (e.g., VirtualBox)<br><sub>❌ Not implemented</sub>                                  |
| **Windows** |   WSL<br><sub>❌ Not implemented</sub><br>\_\_\_<br>Docker<br><sub>✅ Implemented</sub>   |                 ---                  | Docker Desktop (Windows Containers) <br><sub>✅ Implemented</sub><br>\_\_\_<br>VM (Hyper-V)<br><sub>✅ Implemented</sub> |

> <sub>Not implemented</sub> entries indicate solutions not yet implemented in this project. Only solutions marked as **Implemented** are currently available out-of-the-box.

- **Docker**: Used for lightweight Linux container testing on macOS, Linux, and Windows.
- **Windows Containers**: Used for lightweight Windows container testing on Windows hosts with Docker Desktop.
- **Tart VM**: Used for macOS VM testing on macOS hosts.
- **UTM VM**: Used for Windows VM testing on macOS hosts.
- **WSL**: Windows Subsystem for Linux, enables Linux testing on Windows.
- **VM**: Generic virtual machine solutions (e.g., VirtualBox, Hyper-V) for cross-platform testing.
- **Hyper-V**: Used for Windows VM testing on Windows hosts.

> Note: macOS VM testing is only supported on macOS hosts due to Apple licensing restrictions. There might exist workarounds, but they are not covered here.

## System agnostic test approaches

### Local tests for Ubuntu (on linux, WSL, windows or macos)

To test ansible roles for ubuntu, you can use the provided docker-compose setup:

```bash
docker compose -f deployments/docker-compose/ansible/docker-compose.yml up
```

The container will run the ansible playbook within itself. This is a good way to test changes locally without affecting your host system.

To cleanup afterwards, simply run:

```bash
docker compose -f deployments/docker-compose/ansible/docker-compose.yml down
```

## Windows specific test approaches

#### Local tests for Ubuntu on Windows with Hyper-v

To test changes locally on Ubuntu with a Windows host system using Hyper-V, use the Go wrapper workflow from repository root.

Install host dependencies first:

```powershell
go run cmd/main.go install
```

##### Build the Ubuntu box

```powershell
# server
go run cmd/main.go build ubuntu --type server --arch amd64
# desktop
go run cmd/main.go build ubuntu --type desktop --arch amd64
```

##### Create/start the Ubuntu VM

```powershell
$env:VAGRANT_HYPERV_SWITCH = "Default Switch"
go run cmd/main.go create ubuntu --type server --arch amd64
# or desktop
go run cmd/main.go create ubuntu --type desktop --arch amd64
```

##### Provision the Ubuntu VM

```powershell
go run cmd/main.go provision ubuntu --type server --arch amd64 --check
go run cmd/main.go provision ubuntu --type server --arch amd64
```

The command discovers the VM IP automatically and runs Ansible through the Windows/Cygwin wrapper.
Optional Ubuntu provisioning overrides can be set in `.env` using `HYPERV_UBUNTU_ANSIBLE_*`.

For complete manual/advanced steps, see:

- [Ubuntu Packer README](./build/packer/linux/ubuntu/README.md)
- [Ubuntu Hyper-V deployment README](./deployments/vagrant/linux-ubuntu-hyperv/README.md)

### Local tests for windows (on windows)

#### Use Docker Desktop with Windows Containers

To test changes locally on windows, you can use the provided docker-compose setup:

```bash
docker compose -f deployments/docker-compose/ansible-windows/docker-compose.yml up
```

The container will run the ansible playbook against itself.

To cleanup afterwards, run:

```bash
docker compose -f deployments/docker-compose/ansible-windows/docker-compose.yml down
```

Check the [README](deployments/docker-compose/ansible-windows/README.md) in the `deployments/docker-compose/ansible-windows/` folder for more details.

---

#### Use Hyper-V VM

To test changes locally on Windows using Hyper-V, you can create a new virtual machine and configure it to run the Ansible playbook.

Install host dependencies first:

```powershell
go run cmd/main.go install
```

##### Download a Windows .iso file

You will need a Windows .iso file to use as the installation media for your virtual machine. You can download a Windows 10 or Windows Server .iso file from the Microsoft website.

Or use script to download a Windows 11 `.iso` file: [download_win_11.ps1](./scripts/windows/download_win_11.ps1). By default it stores the ISO under the managed cache directory described above.

##### Build a Windows VM

Check [README.md](./build/packer/windows/README.md) for a guide to build a Windows VM with packer and Hyper-V.

##### Run the Windows VM

Check [README.md](./deployments/vagrant/ansible-windows/README.md) for a guide to run the built Windows VM with Vagrant and Hyper-V.

##### Provision the Windows VM (via Go wrapper)

After the VM is running, provision it from the repository root using the unified command:

```bash
go run cmd/main.go provision windows11 --arch amd64 --check
go run cmd/main.go provision windows11 --arch amd64
```

The command discovers the VM IP automatically and runs Ansible through the Windows/Cygwin wrapper.

Set WinRM credentials in project-root `.env` (or process environment):

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

1. `CYGWIN_BASH_PATH` (highest priority)
2. `CYGWIN_TERMINAL_PATH` (used only when `CYGWIN_BASH_PATH` is unset)
3. Auto-detect `C:\tools\cygwin\bin\bash.exe`
4. Auto-detect `C:\cygwin64\bin\bash.exe`

If `CYGWIN_TERMINAL_PATH` points to `mintty.exe`, provisioning resolves it to the sibling `bash.exe`.

## macOS specific test approaches

### Local tests for macOS (on macos)

To test changes locally on macOS, you can use the provided script:

```bash
./scripts/macos/test-ansible-macos.sh
```

The script will run the ansible playbook against a temporary virtual machine managed by Tart.
Tart is a lightweight VM manager for macOS. You can find more information about Tart [here](https://github.com/cirruslabs/tart).

By default, Dev Alchemy uses the Tart image's development credentials for Ansible access (`admin` / `admin`). Override them in `.env` if your image uses different credentials or if you want to avoid relying on the documented defaults:

```bash
TART_MACOS_ANSIBLE_USER=admin
TART_MACOS_ANSIBLE_PASSWORD=admin
```

`TART_MACOS_ANSIBLE_BECOME_PASSWORD` also defaults to `TART_MACOS_ANSIBLE_PASSWORD` when unset.

To cleanup afterwards, run:

```bash
tart delete sequoia-base
```

This will delete the temporary VM.

---

### Local tests for windows (on macos)

On macOS you can use UTM to run a Windows VM for testing ansible changes on windows. UTM is a powerful and easy-to-use virtual machine manager for macOS.
Check [README.md](./build/packer/windows/README.md) for a guide to build a Windows VM with packer and qemu on macos.

Install host dependencies first:

```bash
go run cmd/main.go install
```

You can run the following commands to build and create the Windows 11 VM in UTM:

```bash
# arm64 requires sudo to create a custom .iso file for automated installation.
# sudo rights are evaluated at runtime, so you can run the build command without sudo and it will ask for sudo rights only if needed.
arch=arm64 # or amd64
# sudo go run cmd/main.go build windows11 --arch $arch --headless
go run cmd/main.go build windows11 --arch $arch --headless
# `--headless` applies to `build`, not `create`.
go run cmd/main.go create windows11 --arch $arch
```

Open UTM and start the created Windows VM.

Set WinRM credentials for the provisioning wrapper in project-root `.env` (or process environment):

```dotenv
UTM_WINDOWS_ANSIBLE_USER=Administrator
UTM_WINDOWS_ANSIBLE_PASSWORD=your-secure-password
# Optional (defaults shown):
UTM_WINDOWS_ANSIBLE_CONNECTION=winrm
UTM_WINDOWS_ANSIBLE_WINRM_TRANSPORT=basic
UTM_WINDOWS_ANSIBLE_PORT=5985
```

Now provision the running UTM VM from the repository root:

```bash
go run cmd/main.go provision windows11 --arch $arch --check
go run cmd/main.go provision windows11 --arch $arch
```

The wrapper discovers the VM IP automatically from the generated UTM config and `arp -a`, then runs `ansible-playbook` with an inline inventory target. On macOS it also sets `OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES` for the ansible process automatically.

If you need to inspect the discovered IP manually for troubleshooting:

```bash
bash ./deployments/utm/determine-vm-ip-address.sh --arch $arch --os windows11
```

> ℹ️ Note: newly built Windows images install a dedicated WinRM firewall rule for TCP `5985` on all network profiles, so later NIC or network-profile changes should not break reachability. Older images may still need their network switched to `Private` or an equivalent firewall rule added manually.

---

### Local tests for Ubuntu (on macos)

On macOS you can use UTM to run a Ubuntu VM for testing ansible changes on Ubuntu. UTM is a powerful and easy-to-use virtual machine manager for macOS.

Install host dependencies first:

```bash
go run cmd/main.go install
```

You can run the following commands to build and create the Ubuntu VM in UTM:

```bash
arch=arm64 # or amd64
type=desktop # or server
go run cmd/main.go build ubuntu --arch $arch --type $type
go run cmd/main.go create ubuntu --arch $arch --type $type
```

Open UTM and start the created Ubuntu VM.

Set Ubuntu SSH credentials for the provisioning wrapper in project-root `.env` (or process environment):

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

Now provision the running UTM VM from the repository root:

```bash
go run cmd/main.go provision ubuntu --type $type --arch $arch --check
go run cmd/main.go provision ubuntu --type $type --arch $arch
```

The wrapper discovers the VM IP automatically from the generated UTM config and `arp -a`, then runs `ansible-playbook` with an inline inventory target. On macOS it also sets `OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES` for the ansible process automatically.

If you need to inspect the discovered IP manually for troubleshooting:

```bash
bash ./deployments/utm/determine-vm-ip-address.sh --arch $arch --os "ubuntu-$type"
```

---

## 📦 Supported Tools

Out-of-the-box roles can install (depending on platform):

- java
- jetbrains
- k9s
- kind
- kubectl
- kubelogin
- spotify

> Full list in `roles/` and tagged tasks

---

## 🌍 Cross-Platform Notes

| Platform | Status       | Notes            |
| -------- | ------------ | ---------------- |
| macOS    | ✅ Supported | via Homebrew     |
| Linux    | ✅ Supported | tested on Ubuntu |
| Windows  | ✅ Supported | via cygwin       |

---

## Troubleshooting

- On Windows with cygwin, it can happen that the ansible installation within cygwin is shadowed by another ansible python installation on the windows host. Don't try to install ansible directly on your windows host. Uninstall any other ansible installation and make sure to use the cygwin python installation to install ansible via pip.
- Running Ansible on MacOS can cause forking issues:

```bash
TASK [Gathering Facts] ***************************************************************************************************************************************
objc[9473]: +[NSNumber initialize] may have been in progress in another thread when fork() was called.
objc[9473]: +[NSNumber initialize] may have been in progress in another thread when fork() was called. We cannot safely call it or ignore it in the fork() child process. Crashing instead. Set a breakpoint on objc_initializeAfterForkError to debug.
ERROR! A worker was found in a dead state
```

To avoid this, set the following environment variable before running ansible:

```bash
export OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES
```

## 🤝 Contributing

Contributions welcome! Feel free to:

- Add new roles (e.g., Rust, Java, etc.)
- Improve cross-platform support
- Fix bugs or enhance docs

Review the [CONTRIBUTING.md](./CONTRIBUTING.md) for contribution terms and the Contributor License Agreement (CLA).

---

## 📜 License

Dev Alchemy uses a **dual licensing model**:

### 🚀 Community Edition (Open Source)

The source code is licensed under the **GNU Affero General Public License v3 (AGPLv3)**.

You can review the full text in [LICENSE.md](./LICENSE.md).  
AGPLv3 allows you to use, modify and redistribute the software *as long as* you comply with the AGPLv3 terms.

### 💼 Commercial License

If you wish to use Dev Alchemy in commercial products, SaaS platforms, or closed-source environments without AGPL obligations, a **commercial license** is required.

Commercial license details can be found in [LICENSE_COMMERCIAL.md](./LICENSE_COMMERCIAL.md).

For commercial inquiries, contact:
📧 cc@sautter.cc

### Legacy Licensing

All releases up to and including **v0.1.0** are licensed under the **MIT License**.

---

## 💡 Inspiration

This project was born from a need to simplify dev environment onboarding across multiple systems, without resorting to
OS-specific scripts. With Ansible and a touch of Dev Alchemy, setup becomes reproducible and delightful.

---

🧪 _Happy hacking with `devalchemy`!_
