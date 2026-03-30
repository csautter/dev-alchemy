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

Enabling unencrypted WinRM and Basic authentication increases exposure on the
target machine. Use these settings only in trusted environments or for testing.
For production environments, prefer encrypted connections and stronger
authentication methods, and limit firewall access to trusted networks.

## Option 1: Enable WinRM

```powershell
Set-Item -Path WSMan:\localhost\Service\AllowUnencrypted -Value $true; `
Set-Item -Path WSMan:\localhost\Service\Auth\Basic -Value $true; `
Enable-PSRemoting -Force
```

## Option 2: Enable SSH Server

```powershell
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0; `
Start-Service sshd; Set-Service -Name sshd -StartupType 'Automatic';
```

## Firewall and account requirements

In both cases you may still need to allow inbound connections through the local
firewall. Make sure the account you use for Ansible has administrator
privileges.

If you need a dedicated user for testing, you can create one:

```powershell
# Optional: prefer an existing admin user when possible
net user ansible 'Secret123!@#' /add; `
net localgroup Administrators ansible /add
```

## Running a Windows playbook manually

After remote access is available, you can run the Windows localhost inventory
from the repository root.

Via WinRM:

```powershell
$DevAlchemyPath = "C:\path\to\dev-alchemy"
C:\\cygwin64\\bin\\bash.exe -l -c "cd $DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_winrm.yml -l windows_host"
```

Via SSH:

```powershell
$DevAlchemyPath = "C:\path\to\dev-alchemy"
C:\\cygwin64\\bin\\bash.exe -l -c "cd $DevAlchemyPath && ansible-playbook playbooks/setup.yml -i inventory/localhost_windows_ssh.yml -l windows_host --ask-pass -vvv"
```
