# Windows Ansible Access

Use this guide only when you want to manage a Windows machine over Ansible and
the target does not already expose a supported remote transport.

For most Dev Alchemy onboarding flows, the main [README](../README.md) is the
better starting point. The commands below are mainly for:

- local Ansible runs against the same Windows machine
- existing Windows hosts that need manual remote-access setup
- troubleshooting WinRM or SSH connectivity

## Choose a transport

Windows targets can be reached through either of these transports:

- `WinRM`: more native to Windows, but often needs extra setup and may be
  restricted by company policy
- `SSH`: usually simpler to reason about, but requires the OpenSSH Server
  feature on the target machine

Use only the option you actually need.

## Security note

For localhost runs through `alchemy provision local`, Dev Alchemy now handles a
temporary secure setup for you on Windows. The default WinRM mode creates a
dedicated local admin account with a random password, enables WinRM over HTTPS
on the loopback address for the duration of the run, and restores the prior
WinRM state during cleanup. The SSH alternative
(`alchemy provision local --proto ssh`) creates or updates a temporary local
admin account with a temporary SSH key, enables or installs OpenSSH Server when
needed, sets the default SSH shell to PowerShell for the run, and then restores
the prior SSH state during cleanup.

Manual WinRM setup should also prefer encrypted transport. Avoid unencrypted
WinRM unless you are in a tightly controlled test environment and understand
the exposure you are accepting.

## Option 1: Enable WinRM

For localhost provisioning, prefer:

```powershell
alchemy.exe provision local --check
alchemy.exe provision local
```

For manual `ansible-playbook` use, set up an encrypted WinRM listener and pass
your own credentials and connection variables to Ansible.

## Option 2: Enable SSH Server

For localhost provisioning through the wrapper, prefer:

```powershell
alchemy.exe provision local --proto ssh --check
alchemy.exe provision local --proto ssh --check --yes --force-ssh-uninstall
alchemy.exe provision local --proto ssh
```

For manual setup:

```powershell
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0; `
Start-Service sshd; Set-Service -Name sshd -StartupType 'Automatic';
```

## Firewall and account requirements

In both cases you may still need to allow inbound connections through the local
firewall. Make sure the account you use for Ansible has administrator
privileges.

If you need a dedicated user for testing, you can create one, but for the
localhost wrapper this is handled automatically:

```powershell
# Optional: prefer an existing admin user when possible
net user ansible 'Secret123!@#' /add; `
net localgroup Administrators ansible /add
```

## Running a Windows playbook manually

After remote access is available, use the Windows examples in
[Running Playbooks](./running-playbooks.md).
