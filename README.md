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

### 2. Install Ansible

> Make sure Ansible is installed on your system.

#### macOS (via Homebrew):

```bash
brew install ansible
```

#### Ubuntu / Debian:

```bash
sudo apt update && sudo apt install ansible
```

#### Windows:

For the most native Windows experience, use cygwin and install ansible via pip.

> ⚠️ Make sure to run the commands in an elevated PowerShell (Run as Administrator).<br>

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
choco install -y cygwin --params \"/InstallDir:C:\cygwin64 /NoAdmin /NoStartMenu\"
choco install -y cyg-get
cyg-get python39 python39-pip python39-cryptography openssh git make gcc-core gcc-g++ libffi-devel libssl-devel sshpass
C:\\cygwin64\\bin\\python3.9.exe -m pip install ansible
C:\\cygwin64\\bin\\python3.9.exe -m pip install pywinrm
```

> ℹ️ Instead of using the powershell snippet above, you can also install all windows dependencies with following powershell script:<br> [dev-alchemy-self-setup.ps1](./scripts/windows/dev-alchemy-self-setup.ps1)

Run the powershell script in an elevated PowerShell session (Run as Administrator):

```powershell
./scripts/windows/dev-alchemy-self-setup.ps1
```

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
ansible-playbook playbooks/setup.yml -i inventory/localhost.yml --check
```

```bash
ansible-playbook playbooks/setup.yml -i inventory/localhost.yml
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
C:\\cygwin64\\bin\\bash.exe -l -c "$DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows.yml -l windows_host"
```

Apply the playbook via SSH:

```powershell
$DevAlchemyPath = "C:\path\to\dev-alchemy"
C:\\cygwin64\\bin\\bash.exe -l -c "$DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_ssh.yml -l windows_host --ask-pass -vvv"
```

## 🧩 Structure

```
devalchemy/
├── roles/
│   ├── role/
│   ├── role2/
│   └── role3/
├── inventory/
│   └── localhost.yml
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

| Host OS     |                                      Test Linux                                       |              Test macOS              |                                                       Test Windows                                                       |
| ----------- | :-----------------------------------------------------------------------------------: | :----------------------------------: | :----------------------------------------------------------------------------------------------------------------------: |
| **macOS**   |                          Docker<br><sub>✅ Implemented</sub>                          | Tart VM<br><sub>✅ Implemented</sub> |                                         UTM Qemu VM<br><sub>✅ Implemented</sub>                                         |
| **Linux**   |                          Docker<br><sub>✅ Implemented</sub>                          |                 ---                  |                                  VM (e.g., VirtualBox)<br><sub>❌ Not implemented</sub>                                  |
| **Windows** | WSL<br><sub>❌ Not implemented</sub><br>\_\_\_<br>Docker<br><sub>✅ Implemented</sub> |                 ---                  | Docker Desktop (Windows Containers) <br><sub>✅ Implemented</sub><br>\_\_\_<br>VM (Hyper-V)<br><sub>✅ Implemented</sub> |

> <sub>Not implemented</sub> entries indicate solutions not yet implemented in this project. Only solutions marked as **Implemented** are currently available out-of-the-box.

- **Docker**: Used for lightweight Linux container testing on macOS, Linux, and Windows.
- **Windows Containers**: Used for lightweight Windows container testing on Windows hosts with Docker Desktop.
- **Tart VM**: Used for macOS VM testing on macOS hosts.
- **UTM VM**: Used for Windows VM testing on macOS hosts.
- **WSL**: Windows Subsystem for Linux, enables Linux testing on Windows.
- **VM**: Generic virtual machine solutions (e.g., VirtualBox, Hyper-V) for cross-platform testing.
- **Hyper-V**: Used for Windows VM testing on Windows hosts.

> Note: macOS VM testing is only supported on macOS hosts due to Apple licensing restrictions. There might exist workarounds, but they are not covered here.

### Local tests for ubuntu (on linux, WSL, windows or macos)

To test ansible roles for ubuntu, you can use the provided docker-compose setup:

```bash
docker compose -f deployments/docker-compose/ansible/docker-compose.yml up
```

The container will run the ansible playbook within itself. This is a good way to test changes locally without affecting your host system.

To cleanup afterwards, simply run:

```bash
docker compose -f deployments/docker-compose/ansible/docker-compose.yml down
```

### Local tests for macOS (on macos)

To test changes locally on macOS, you can use the provided script:

```bash
./scripts/macos/test-ansible-macos.sh
```

The script will run the ansible playbook against a temporary virtual machine managed by Tart.
Tart is a lightweight VM manager for macOS. You can find more information about Tart [here](https://github.com/cirruslabs/tart).

To cleanup afterwards, run:

```bash
tart delete sequoia-base
```

This will delete the temporary VM.

---

### Local tests for windows (on macos)

On macOS you can use UTM to run a Windows VM for testing ansible changes on windows. UTM is a powerful and easy-to-use virtual machine manager for macOS.
Check [README.md](./build/packer/windows/README.md) for a guide to build a Windows VM with packer and qemu on macos.

After the VM is built, you can add it to UTM and start it. This step is currently automated just for Windows arm64. For x86_64 you need to add the VM manually to UTM.
See script [create-windows11-utm-vm.sh](./deployments/utm/create-windows11-utm-vm.sh) for details about the UTM VM creation.
You can use following scripts to create the UTM VM and determine its IP address:

```bash
bash ./deployments/utm/create-windows11-utm-vm.sh
bash ./deployments/utm/determine-vm-ip-address.sh
```

You can create the inventory file using a Bash script and the `$vagrant_ip` variable:

```bash
vagrant_ip="YOUR_VM_IP_HERE"
cat <<EOF > ./inventory/utm_windows_winrm.yml
all:
  children:
    windows:
      hosts:
        windows_host:
          ansible_host: $vagrant_ip
          ansible_user: Administrator
          ansible_password: P@ssw0rd!
          ansible_connection: winrm
          ansible_winrm_transport: basic
          ansible_port: 5985
EOF
```

Now you can run the ansible playbook against the UTM Windows VM:

```bash
export OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES
ansible-playbook ./playbooks/setup.yml -i ./inventory/utm_windows_winrm.yml -l windows_host -vvv --check
ansible-playbook ./playbooks/setup.yml -i ./inventory/utm_windows_winrm.yml -l windows_host -vvv
```

> ℹ️ Note: there is a known issue, that ansible might fail to connect via winrm when the VM has configured the Network interface as Public network. Switching it to Private network should resolve the issue.

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

#### Use Hyper-V VM

To test changes locally on Windows using Hyper-V, you can create a new virtual machine and configure it to run the Ansible playbook.

##### Download a Windows .iso file

You will need a Windows .iso file to use as the installation media for your virtual machine. You can download a Windows 10 or Windows Server .iso file from the Microsoft website.

Or use script to download a Windows 11 .iso file: [download_win_11.ps1](./scripts/windows/download_win_11.ps1)

##### Build a Windows VM

Check [README.md](./build/packer/windows/README.md) for a guide to build a Windows VM with packer and Hyper-V.

##### Run the Windows VM

Check [README.md](./deployments/vagrant/ansible-windows/README.md) for a guide to run the built Windows VM with Vagrant and Hyper-V.

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

## 🤝 Contributing

Contributions welcome! Feel free to:

- Add new roles (e.g., Rust, Java, etc.)
- Improve cross-platform support
- Fix bugs or enhance docs

---

## 📜 License

MIT License — see [LICENSE](LICENSE) file.

---

## 💡 Inspiration

This project was born from a need to simplify dev environment onboarding across multiple systems, without resorting to
OS-specific scripts. With Ansible and a touch of Dev Alchemy, setup becomes reproducible and delightful.

---

🧪 _Happy hacking with `devalchemy`!_
