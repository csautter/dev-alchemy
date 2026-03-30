# 🧪 devalchemy

**devalchemy** is an opinionated, cross-platform orchestration toolkit for
development and non-development environments. It combines
[Ansible](https://www.ansible.com/) with OS-specific system tools to provide a
unified way to test, provision, and manage machines across **macOS**,
**Linux**, and **Windows**.

Built for small teams working with heterogeneous device fleets,
**devalchemy** addresses a costly and familiar problem: developers lose hours
setting up machines, guessing how a system is supposed to work, and applying
OS-specific hacks just to reach a usable baseline. It turns both fresh and
existing machines into consistent, reproducible developer setups across
**macOS**, **Linux**, and **Windows**.

It is not intended to replace classic centralized UEM/MDM solutions. Instead, it
starts where those tools usually stop by managing developer tooling,
workflows, and team-specific environment standards. With its
infrastructure-as-code approach, **devalchemy** helps teams reduce onboarding
time, lower recurring support effort, and keep setups maintainable through
unified standards, version control, and CI/CD. It can also be integrated into
existing environments, including more complex scenarios such as consultants
working within client-managed setups.

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

**devalchemy** grew out of repeated onboarding and support pain in
cross-platform teams. When every operating system needs different workarounds,
setup knowledge becomes fragmented, failures are harder to reproduce, and
senior team members spend too much time helping others get unstuck. The goal
is to standardize those workflows so teams spend less time fixing machines and
more time delivering work.

Key problems addressed:

- Developers lose time figuring out how each machine should be configured
- Inconsistent setups across different operating systems
- Slow onboarding for new team members
- High support effort when setup issues are hard to reproduce
- Environment standards that drift and become harder to maintain
- Over-reliance on OS-specific scripts and manual fixes
- Security concerns from using elevated privileges more often than necessary

## Base Concepts

The core idea of **devalchemy** is to use Ansible playbooks and roles to define and automate the setup of development environments. This includes installing essential tools, configuring settings, and managing dependencies.<br>
Every role is platform independent and can be applied to macOS, Linux, and Windows. The playbooks are designed to be modular, allowing users to pick and choose which components they want to install.<br>
The setup is idempotent, meaning you can run the playbooks multiple times without causing issues or duplications. This ensures that your development environment remains consistent and up-to-date.<br>
Despite the common use of Ansible in server environments where changes are **pushed** from a central location, **devalchemy** is designed for local **pull** based execution on individual machines. This approach allows developers to maintain control over their own environments while still benefiting from automation. Every ansible run can be simulated with `--check` to see what changes would be applied.

---

## 🚀 Getting Started

### 1. Download a Release Binary

Release assets are published on the
[GitHub Releases page](https://github.com/csautter/dev-alchemy/releases) using the pattern
`dev-alchemy_<version>_<os>_<arch>`, with `.tar.gz` archives for macOS/Linux and `.zip`
archives for Windows.

After extraction, the executable is named `alchemy` on macOS/Linux and `alchemy.exe` on Windows.

The examples below resolve the latest published release tag automatically before downloading
the matching archive for the selected platform.

#### macOS / Linux example

```bash
TAG="$(curl -fsSL https://api.github.com/repos/csautter/dev-alchemy/releases/latest | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
VERSION="${TAG#v}"
curl -fLO "https://github.com/csautter/dev-alchemy/releases/download/${TAG}/dev-alchemy_${VERSION}_linux_amd64.tar.gz"
tar -xzf "dev-alchemy_${VERSION}_linux_amd64.tar.gz"
chmod +x ./alchemy
./alchemy build list
```

#### Windows example

```powershell
$Release = Invoke-RestMethod "https://api.github.com/repos/csautter/dev-alchemy/releases/latest"
$Tag = $Release.tag_name
$Version = $Tag.TrimStart("v")
Invoke-WebRequest -OutFile "dev-alchemy_${Version}_windows_amd64.zip" "https://github.com/csautter/dev-alchemy/releases/download/$Tag/dev-alchemy_${Version}_windows_amd64.zip"
Expand-Archive "dev-alchemy_${Version}_windows_amd64.zip" -DestinationPath .
.\alchemy.exe build list
```

When you run a release binary outside a Git checkout, Dev Alchemy extracts its embedded
runtime assets into the managed app-data directory and executes from there.

### 2. Clone the repo for development or to run playbooks directly from the repository:

```bash
git clone https://github.com/csautter/dev-alchemy.git
cd dev-alchemy
```

### 3. Install Host Dependencies

#### macOS

```bash
alchemy install
```

This runs [scripts/macos/dev-alchemy-install-dependencies.sh](./scripts/macos/dev-alchemy-install-dependencies.sh).

#### Ubuntu / Debian

The `install` command is currently intended for macOS and Windows hosts. On Linux, install Ansible manually:

```bash
sudo apt update && sudo apt install ansible
```

#### Windows

Run the command in an elevated PowerShell session (Run as Administrator):

```powershell
alchemy.exe install
```

This runs [scripts/windows/dev-alchemy-self-setup.ps1](./scripts/windows/dev-alchemy-self-setup.ps1).

To force a VM rebuild even when the cached build artifact already exists, use:

```powershell
alchemy.exe build windows11 --arch amd64 --no-cache
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
- `project/` for the embedded runtime project used by standalone binaries outside a Git checkout

You can override the default location by setting `DEV_ALCHEMY_APP_DATA_DIR`. Dev Alchemy also exports `DEV_ALCHEMY_CACHE_DIR`, `DEV_ALCHEMY_VAGRANT_DIR`, and `DEV_ALCHEMY_PACKER_CACHE_DIR` for helper scripts and manual workflows.

On the first standalone run, Dev Alchemy extracts bundled scripts, playbooks, and other
runtime assets into `DEV_ALCHEMY_APP_DATA_DIR/project`. Later runs keep that managed
tree in sync so the standalone `alchemy` binary can operate without a repository checkout.



##### Windows remote access

Windows hosts usually need either `WinRM` or `SSH` enabled before Ansible can
manage them, including some localhost-style runs on the same machine.

The setup commands, security notes, and manual Windows playbook examples live in the dedicated guide:

- [Windows Ansible Access](./docs/windows-ansible-access.md)

---

### 4. Discover Available Targets and Commands

Use the `list` subcommands to see what the current host can build, create, start, provision, stop, or destroy before running a longer workflow:

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
alchemy build --help
alchemy provision --help
```

---

### 5. Run the Playbook

The manual `ansible-playbook` workflows for localhost, remote hosts,
VMs, and Windows live in the dedicated guide:

- [Running Playbooks](./docs/running-playbooks.md)

If the Windows target does not already have remote access configured, start
with [Windows Ansible Access](./docs/windows-ansible-access.md).

## Testing

### 🧪 Cross-Platform Testing Matrix

| Host OS     |                                        Test Linux                                         |              Test macOS              |                                                       Test Windows                                                       |
| ----------- | :---------------------------------------------------------------------------------------: | :----------------------------------: | :----------------------------------------------------------------------------------------------------------------------: |
| **macOS**   | Docker<br><sub>✅ Implemented</sub><br>\_\_\_<br>UTM Qemu VM<br><sub>✅ Implemented</sub> | Tart VM<br><sub>✅ Implemented</sub> |                                         UTM Qemu VM<br><sub>✅ Implemented</sub>                                         |
| **Linux**   |                            Docker<br><sub>✅ Implemented</sub>                            |                 ---                  |                                  VM (e.g., VirtualBox)<br><sub>❌ Not implemented</sub>                                  |
| **Windows** |   WSL<br><sub>❌ Not implemented</sub><br>\_\_\_<br>Docker<br><sub>✅ Implemented</sub>   |                 ---                  | Docker Desktop (Windows Containers) <br><sub>✅ Implemented</sub><br>\_\_\_<br>VM (Hyper-V)<br><sub>✅ Implemented</sub><br>\_\_\_<br>VM (VirtualBox)<br><sub>⚠️ Unstable</sub> |

> <sub>Not implemented</sub> entries indicate solutions not yet implemented in this project. Only solutions marked as **Implemented** are currently available out-of-the-box.

- **Docker**: Used for lightweight Linux container testing on macOS, Linux, and Windows.
- **Windows Containers**: Used for lightweight Windows container testing on Windows hosts with Docker Desktop.
- **Tart VM**: Used for macOS VM testing on macOS hosts.
- **UTM VM**: Used for Windows VM testing on macOS hosts.
- **WSL**: Windows Subsystem for Linux, enables Linux testing on Windows.
- **VM**: Generic virtual machine solutions (e.g., VirtualBox, Hyper-V) for cross-platform testing.
- **Hyper-V**: Used for Windows VM testing on Windows hosts.

> Note: macOS VM testing is only supported on macOS hosts due to Apple licensing restrictions. There might exist workarounds, but they are not covered here.

### Workflow

Most testing flows now go through the same VM lifecycle interface:

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

Examples:

```bash
alchemy build ubuntu --type server --arch amd64
alchemy create ubuntu --type server --arch amd64
alchemy provision ubuntu --type server --arch amd64 --check

alchemy stop ubuntu --type server --arch amd64
alchemy start ubuntu --type server --arch amd64
```

For manual playbook commands, platform-specific testing examples, environment
variables, Docker-based flows, and troubleshooting commands, see:

- [Running Playbooks](./docs/running-playbooks.md)
- [Testing Workflows](./docs/testing-workflows.md)
- [Windows Ansible Access](./docs/windows-ansible-access.md)

## 📦 Example Roles

The repository includes a growing set of example Ansible roles. The catalog,
repository layout overview, and quick customization examples live in:

- [Example Ansible Roles](./docs/example-roles.md)

For the source of truth, inspect [`roles/`](./roles/) and the relevant tagged
tasks.

---

## Troubleshooting

Rare, host-specific issues are documented separately:

- [Troubleshooting Guide](./docs/troubleshooting.md)

## 🤝 Contributing

Contributions welcome! Feel free to:

- Add new roles (e.g., Rust, Java, etc.)
- Improve cross-platform support
- Fix bugs or enhance docs

Review the [CONTRIBUTING.md](./CONTRIBUTING.md) for contribution terms and the Contributor License Agreement (CLA).

---

## 📜 License

Dev Alchemy is available under a dual-licensing model:

### Open Source

The community edition is licensed under the **GNU Affero General Public License v3 (AGPLv3)**.

See [LICENSE.md](./LICENSE.md) for the full license text.

### Commercial Use

If you need to use Dev Alchemy in commercial products, SaaS platforms, or other closed-source environments without AGPL obligations, a separate commercial license is available.

See [LICENSE_COMMERCIAL.md](./LICENSE_COMMERCIAL.md) for commercial licensing terms.

For commercial inquiries, contact:
📧 cc@sautter.cc

### Historical Note

Early releases were published under the **MIT License**.
If you are using an older tag or release, refer to the license file included with that version for the exact terms that apply.

---

## 💡 Inspiration

This project was born from a need to simplify dev environment onboarding across multiple systems, without resorting to
OS-specific scripts. With Ansible and a touch of Dev Alchemy, setup becomes reproducible and delightful.

---

🧪 _Happy hacking with `devalchemy`!_
