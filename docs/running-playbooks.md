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
alchemy provision local
```

`alchemy provision local` uses `inventory/localhost.yaml` on macOS/Linux and
`inventory/localhost_windows_winrm.yml` on Windows. On Windows the wrapper
creates a temporary administrator account with a random password, enables
encrypted WinRM over HTTPS for the run, and then restores the WinRM state while
disabling the temporary account during cleanup. The macOS/Linux local target is
currently marked unstable until it has been validated end-to-end.

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
$DevAlchemyPath = "C:\path\to\dev-alchemy"
C:\\cygwin64\\bin\\bash.exe -l -c "cd $DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_winrm.yml -l windows_host"
```

### Via SSH

```powershell
$DevAlchemyPath = "C:\path\to\dev-alchemy"
C:\\cygwin64\\bin\\bash.exe -l -c "cd $DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_ssh.yml -l windows_host --ask-pass -vvv"
```
