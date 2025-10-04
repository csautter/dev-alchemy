# Run Ansible Windows with Vagrant and Hyper-V

This guide will help you set up and run Ansible playbooks on a Windows VM using Vagrant with Hyper-V as the provider.
All commands are meant to be run in a powershell terminal on a Windows host machine.

## Prerequisites

Use the [dev-alchemy-self-setup.ps1](../../../scripts/windows/dev-alchemy-self-setup.ps1) script to set up your Windows host machine with the necessary tools and configurations.

Ensure you have the following installed:

- [Vagrant](https://www.vagrantup.com/downloads)
- [Hyper-V](https://docs.microsoft.com/en-us/virtualization/hyper-v-on-windows/quick-start/enable-hyper-v)
- [Cygwin](https://www.cygwin.com/install.html)
- [Ansible](https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html) (via Cygwin)

## Adding the Vagrant Box

Load the Vagrant box and start the VM using Hyper-V as the provider:

```bash
vagrant box add win11-packer .\vendor\windows\win11-hyperv.box --provider hyperv
vagrant up --provider hyperv
```

After the VM is up, you can connect to it using Hyper-V Manager or via RDP. The default credentials are:

- Username: `Administrator`
- Password: `P@ssw0rd!`

## Determine the IP Address of the Vagrant Box

You can find the IP address of the Vagrant box using the following command:

```bash
vagrant winrm -c "ipconfig"
```

Alternatively, you can use PowerShell to extract the IP address directly:

```powershell
$vagrant_ip = (vagrant winrm -c "ipconfig" | Select-String -Pattern 'IPv4 Address.*: (\d{1,3}\.){3}\d{1,3}' | ForEach-Object { $_.Matches[0].Value.Split(':')[1].Trim() })
Write-Output "Vagrant Box IP Address: $vagrant_ip"
```

## Write the Inventory File

You can create the inventory file using PowerShell and the `$vagrant_ip` variable:

```powershell
$inventory = @"
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
"@
$inventory | Set-Content -Path "./inventory/hyperv_windows_winrm.yml"
Write-Output "Inventory file created at ./inventory/hyperv_windows_winrm.yml"
```

## Run Ansible Playbook

Run the Ansible playbook using the created inventory file. Make sure to replace `<path-to-your-repo>` with the actual path to your repository.
On Windows, you need to use Cygwin to run Ansible commands.

```bash
# start a Cygwin bash session
cygwin bash
# navigate to the dev-alchemy directory
cd /cygdrive/c/<path-to-your-repo>/dev-alchemy/
ansible-playbook ./playbooks/setup.yml -i ./inventory/hyperv_windows_winrm.yml -l windows_host -vvv --check
ansible-playbook ./playbooks/setup.yml -i ./inventory/hyperv_windows_winrm.yml -l windows_host -vvv
```

## Destroying the Vagrant Box

When you are done, you can destroy the Vagrant box and remove the box from your system:

```bash
vagrant destroy
vagrant box remove win11-packer --provider hyperv
```
