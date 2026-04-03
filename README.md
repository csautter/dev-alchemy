# 🧪 devalchemy

**devalchemy** is an opinionated cross-platform automation toolkit for setting
up, testing, and maintaining developer environments on **macOS**, **Linux**,
and **Windows**. It combines [Ansible](https://www.ansible.com/) with
host-specific VM and system tooling so teams can manage local machines, remote
hosts, and disposable test systems through one repository.

It is especially useful when a team has a mixed device fleet and wants one
repeatable way to:

- onboard developers faster
- keep machine setup consistent across operating systems
- test provisioning changes safely before rolling them out
- reduce one-off scripts, tribal knowledge, and repeated support work

Dev Alchemy is not a replacement for classic MDM/UEM tooling. It complements
those tools by handling the developer-tooling and workflow layer that often
remains manual, team-specific, and hard to reproduce.

> _"Transform your system into a dev powerhouse with a touch of automation magic."_

## Why It Helps

Without a shared automation baseline, cross-platform teams usually run into the
same problems:

- developers spend hours figuring out how a machine is supposed to look
- onboarding depends too much on senior team members
- setup differences become hard to debug and reproduce
- OS-specific scripts drift over time

Dev Alchemy addresses that by keeping setup logic in versioned Ansible roles
and playbooks, then using host-appropriate tooling to apply and test them.

## Support Snapshot

The project currently supports these host-to-target workflows:

| Host OS | What you can automate today |
| --- | --- |
| **macOS** | Managed workflows for **macOS** (Tart), **Ubuntu** (UTM), and **Windows 11** (UTM) |
| **Windows** | Managed workflows for **Ubuntu** (Hyper-V) and **Windows 11** (Hyper-V, plus VirtualBox as unstable) |
| **Linux** | Direct Ansible runs and Docker-based Linux testing; managed VM workflows are more limited today |

This means Dev Alchemy can cover every currently supported guest OS family on a
**macOS host**, and every currently supported family except **macOS** on a
**Windows host**.

Use the built-in discovery commands to see the exact combinations available on
your current machine:

```bash
alchemy build list
alchemy create list
alchemy start list
alchemy provision list
alchemy stop list
alchemy destroy list
```

> Note: macOS guests are only supported on macOS hosts due to Apple platform
> and licensing restrictions.

## Base Model

Dev Alchemy follows a few simple ideas:

- **Ansible roles and playbooks are the source of truth** for machine setup
- **roles stay cross-platform where possible**, with OS-specific handling where needed
- **runs are idempotent**, so the same workflow can be applied repeatedly
- **execution is pull-oriented**, so machines can run their own automation locally
- **`--check` mode matters**, so changes can be previewed before they apply

That makes it practical both for daily developer use and for testing changes in
VMs before applying them to real machines.

You can also use the built-in wrapper for host-local provisioning:

```bash
alchemy provision local --check
alchemy provision local --check --yes
alchemy provision local --check --yes --force-winrm-uninstall
alchemy provision local
```

On Windows this uses the documented localhost WinRM inventory. On macOS and
Linux it uses the standard localhost inventory. On Windows the wrapper
creates a temporary administrator account with a random password, enables
encrypted WinRM over HTTPS for the run, and then disables the temporary account
again during cleanup. Because those are significant host changes, the Windows
local flow asks for confirmation by default; use `--yes` when you need to run
it non-interactively. Windows will also show a UAC prompt before the privileged
bootstrap and cleanup steps run. Use
`--force-winrm-uninstall` to force uninstall and disable of WinRM. The macOS/Linux local target is
currently marked unstable until it has been validated end-to-end.

## 🚀 Getting Started

### 1. Choose how you want to run it

You can either:

- download a release binary for normal use
- clone the repository when you want to edit playbooks, roles, or project code

#### Download a release binary

Release assets are published on the
[GitHub Releases page](https://github.com/csautter/dev-alchemy/releases) as
`dev-alchemy_<version>_<os>_<arch>`.

After extraction, the executable is named `alchemy` on macOS/Linux and
`alchemy.exe` on Windows.

macOS / Linux example:

```bash
TAG="$(curl -fsSL https://api.github.com/repos/csautter/dev-alchemy/releases/latest | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
VERSION="${TAG#v}"
curl -fLO "https://github.com/csautter/dev-alchemy/releases/download/${TAG}/dev-alchemy_${VERSION}_linux_amd64.tar.gz"
tar -xzf "dev-alchemy_${VERSION}_linux_amd64.tar.gz"
chmod +x ./alchemy
./alchemy build list
```

Windows example:

```powershell
$Release = Invoke-RestMethod "https://api.github.com/repos/csautter/dev-alchemy/releases/latest"
$Tag = $Release.tag_name
$Version = $Tag.TrimStart("v")
Invoke-WebRequest -OutFile "dev-alchemy_${Version}_windows_amd64.zip" "https://github.com/csautter/dev-alchemy/releases/download/$Tag/dev-alchemy_${Version}_windows_amd64.zip"
Expand-Archive "dev-alchemy_${Version}_windows_amd64.zip" -DestinationPath .
.\alchemy.exe build list
```

When you run a release binary outside a Git checkout, Dev Alchemy extracts its
embedded runtime assets into a managed app-data directory. See
[Managed Application Data](./docs/managed-application-data.md) for the default
locations and override options.

#### Clone the repository

```bash
git clone https://github.com/csautter/dev-alchemy.git
cd dev-alchemy
```

### 2. Install host dependencies

#### macOS

```bash
alchemy install
```

This runs
[scripts/macos/dev-alchemy-install-dependencies.sh](./scripts/macos/dev-alchemy-install-dependencies.sh).

#### Ubuntu / Debian

The `install` command is currently intended for macOS and Windows hosts. On
Linux, install Ansible manually:

```bash
sudo apt update && sudo apt install ansible
```

#### Windows

Run the command in an elevated PowerShell session:

```powershell
alchemy.exe install
```

This runs
[scripts/windows/dev-alchemy-self-setup.ps1](./scripts/windows/dev-alchemy-self-setup.ps1).

### 3. Discover what your host supports

Start with the `list` commands before running a longer workflow:

```bash
alchemy build list
alchemy create list
alchemy provision list
```

Use `--help` when you want the supported flags for a command:

```bash
alchemy --help
alchemy build --help
alchemy provision --help
```

### 4. Run your first useful workflow

There are two common entry paths.

#### A. Configure the current machine directly

Use the built-in wrapper first when you want the shared command surface:

```bash
alchemy provision local --check
alchemy provision local
```

For the underlying direct `ansible-playbook` commands from the repository root:

```bash
ansible-playbook playbooks/setup.yml -i inventory/localhost.yaml --check
ansible-playbook playbooks/setup.yml -i inventory/localhost.yaml
```

For Windows localhost or remote-target examples, use
[Running Playbooks](./docs/running-playbooks.md).

#### B. Test the setup in a disposable VM first

Example on a supported host:

```bash
alchemy build ubuntu --type server --arch amd64
alchemy create ubuntu --type server --arch amd64
alchemy provision ubuntu --type server --arch amd64 --check
alchemy provision ubuntu --type server --arch amd64
```

If you are targeting Windows and remote access is not configured yet, start
with [Windows Ansible Access](./docs/windows-ansible-access.md).

## Docs Map

The root README is the fast entry point. Use these guides when you want the
next level of detail:

- [Running Playbooks](./docs/running-playbooks.md) for localhost, remote-host,
  VM, and Windows `ansible-playbook` examples
- [Testing Workflows](./docs/testing-workflows.md) for host-specific VM and
  Docker test flows
- [Managed Application Data](./docs/managed-application-data.md) for cache,
  runtime, and app-data locations
- [Windows Ansible Access](./docs/windows-ansible-access.md) for WinRM and SSH
  setup on Windows targets
- [Example Ansible Roles](./docs/example-roles.md) for the current sample role
  catalog and repository layout
- [Troubleshooting Guide](./docs/troubleshooting.md) for rare host-specific
  issues

## 📦 Example Roles

The repository already includes example roles for common developer tooling such
as `brew`, `java`, `jetbrains`, `k9s`, `kind`, `kubectl`, `kubelogin`,
`openssh`, `python`, and `spotify`.

Use [Example Ansible Roles](./docs/example-roles.md) as the catalog and
[`roles/`](./roles/) as the source of truth.

## 🤝 Contributing

Contributions are welcome. Good areas to improve include:

- more roles and playbooks
- broader cross-platform coverage
- better docs and troubleshooting guidance
- bug fixes and test coverage

See [CONTRIBUTING.md](./CONTRIBUTING.md) for contribution terms and the
Contributor License Agreement (CLA).

## 📜 License

Dev Alchemy uses a dual-licensing model.

### Open Source

The community edition is licensed under the **GNU Affero General Public License
v3 (AGPLv3)**.

See [LICENSE.md](./LICENSE.md).

### Commercial Use

If you need to use Dev Alchemy in commercial products, SaaS platforms, or
other closed-source environments without AGPL obligations, a separate
commercial license is available.

See [LICENSE_COMMERCIAL.md](./LICENSE_COMMERCIAL.md).

For commercial inquiries: `cc@sautter.cc`

### Historical Note

Early releases were published under the **MIT License**. If you use an older
tag or release, refer to the license file included with that version.

## 💡 Inspiration

This project grew out of real onboarding and support pain in mixed-OS teams.
The goal is simple: make machine setup reproducible, testable, and much less
dependent on memory or hand-written host-specific scripts.
