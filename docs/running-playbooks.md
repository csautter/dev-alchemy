# Running Playbooks

Use the root README for installation and initial setup, then use the commands
below when you want to run `playbooks/setup.yml` directly from the repository.

## Before you run a playbook

- Run the commands from the repository root.
- Install host dependencies first as described in the root
  [README](../README.md).
- Use `--check` first when you want a dry run before applying changes.
- Adjust the inventory or pass extra variables on the CLI as needed for your
  environment.

## Run on localhost

CLI wrapper:

```bash
alchemy provision local --check
alchemy provision local --proto ssh --check
alchemy provision local --proto ssh --check --yes --force-ssh-uninstall
alchemy provision local --playbook ./playbooks/bootstrap.yml
alchemy provision local -- --diff --tags java
alchemy provision local --inventory-path ./inventory/remote.yml -- --limit workstation --ask-become-pass
alchemy provision local
```

`alchemy provision local` uses `inventory/localhost.yaml` on macOS/Linux and
`inventory/localhost_windows_winrm.yml` on Windows by default. On Windows,
`--proto ssh` switches the wrapper to `inventory/localhost_windows_ssh.yml`.
The WinRM wrapper creates a temporary administrator account with a random
password, enables encrypted WinRM over HTTPS on the loopback address for the
run, and then restores the WinRM state during cleanup. The SSH wrapper creates
or updates a temporary administrator account with a temporary SSH key, enables
or installs OpenSSH Server when needed, sets the default SSH shell to
PowerShell for the run, and then restores the prior SSH service, firewall,
authorized_keys, and shell state afterwards. If the wrapper had to install
OpenSSH Server for the run, cleanup disables `sshd` but leaves the OpenSSH
Server capability installed so cleanup does not require a reboot. If
`devalchemy_ansible` already exists, the wrapper reuses it and rotates its
password for the run; cleanup does not restore the previous password, so that
account should be considered automation-managed. The macOS/Linux local target
is currently marked unstable until it has been validated end-to-end. Extra
`ansible-playbook` flags can be passed after `--`, `--inventory-path`
overrides the default local inventory file, and `--playbook` overrides the
default playbook path.
`--force-winrm-uninstall` only applies to the default WinRM mode, while
`--force-ssh-uninstall` only applies to `--proto ssh`. It still does not
uninstall OpenSSH Server; if you need to roll that back after a run, use the
manual OpenSSH removal steps in
[`windows-ansible-access.md`](./windows-ansible-access.md#remove-openssh-server-after-a-wrapper-run).

Dry run:

```bash
ansible-playbook playbooks/setup.yml -i inventory/localhost.yaml --check
```

Apply:

```bash
ansible-playbook playbooks/setup.yml -i inventory/localhost.yaml
```

## Run on a remote host or in a VM

Dry run:

```bash
HOST="192.168.179.21"
cat <<EOF > inventory/remote.yml
all:
  hosts:
    $HOST:
      ansible_user: admin
EOF
ansible-playbook playbooks/setup.yml -i inventory/remote.yml -l "$HOST" --ask-pass --ask-become-pass --check
```

Apply:

```bash
ansible-playbook playbooks/setup.yml -i inventory/remote.yml -l "$HOST" --ask-pass --ask-become-pass
```

## Run on Windows

If the Windows target does not already have remote access configured, start
with [Windows Ansible Access](./windows-ansible-access.md).

### Via WinRM

For localhost runs, prefer `alchemy provision local`. It bootstraps the secure
temporary WinRM account automatically.

If you run `ansible-playbook` directly with
`inventory/localhost_windows_winrm.yml`, you are responsible for supplying your
own secure WinRM connection variables and credentials.

```powershell
alchemy.exe provision local --check
alchemy.exe provision local
```

### Via SSH

For localhost runs, prefer:

```powershell
alchemy.exe provision local --proto ssh --check
alchemy.exe provision local --proto ssh
```

The wrapper-owned `devalchemy_ansible` account may be reused across runs. When
it is reused, the bootstrap rotates its password and cleanup does not restore
the previous value. If the wrapper had to install OpenSSH Server, cleanup
leaves that capability installed and disables `sshd` instead of uninstalling
the capability.

If you run `ansible-playbook` directly with
`inventory/localhost_windows_ssh.yml`, you are responsible for supplying your
own secure SSH user, key, and shell variables.

```powershell
$DevAlchemyPath = "C:\path\to\dev-alchemy"
C:\\cygwin64\\bin\\bash.exe -l -c "cd $DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_ssh.yml -l windows_host -e ansible_user=admin -e ansible_ssh_private_key_file=/path/to/key -e ansible_shell_type=powershell -e ansible_shell_executable=powershell.exe -vvv"
```
