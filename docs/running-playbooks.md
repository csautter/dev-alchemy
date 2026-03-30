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

```powershell
$DevAlchemyPath = "C:\path\to\dev-alchemy"
C:\\cygwin64\\bin\\bash.exe -l -c "cd $DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_winrm.yml -l windows_host"
```

### Via SSH

```powershell
$DevAlchemyPath = "C:\path\to\dev-alchemy"
C:\\cygwin64\\bin\\bash.exe -l -c "cd $DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_ssh.yml -l windows_host --ask-pass -vvv"
```
