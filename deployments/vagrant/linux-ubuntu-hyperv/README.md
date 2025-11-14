# Run Ubuntu Linux with Vagrant and Hyper-V

This guide will help you set up and run Ansible playbooks on a Ubuntu Linux VM using Vagrant with Hyper-V as the provider.
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
vagrant box add linux-ubuntu-packer .\internal\vagrant\linux-ubuntu-hyperv.box --provider hyperv
cd deployments\vagrant\linux-ubuntu-hyperv
vagrant up --provider hyperv
cd ..\..\..
```

After the VM is up, you can connect to it using Hyper-V Manager or via SSH. The default credentials are:

- Username: `packer`
- Password: `P@ssw0rd!`

## Determine the IP Address of the Vagrant Box

You can find the IP address of the Vagrant box using the following command:

```bash
vagrant ssh -c "ip a"
```

Alternatively, you can use PowerShell to extract the IP address directly:

```powershell
$vagrant_ip = (vagrant ssh -c "hostname -I" | Select-Object -First 1).Trim()
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
        ubuntu_host:
          ansible_host: $vagrant_ip
          ansible_user: packer
          ansible_password: P@ssw0rd!
          ansible_ssh_common_args: "-o StrictHostKeyChecking=no -o ServerAliveInterval=10 -o ServerAliveCountMax=3 -o ControlMaster=no -o ControlPersist=no"
          ansible_ssh_timeout: 120 # timeout in seconds
          ansible_ssh_retries: 3
          ansible_become_password: P@ssw0rd!
"@
$inventory | Set-Content -Path "./inventory/hyperv_ubuntu.yml"
Write-Output "Inventory file created at ./inventory/hyperv_ubuntu.yml"
```

## Run Ansible Playbook

Run the Ansible playbook using the created inventory file. Make sure to replace `<path-to-your-repo>` with the actual path to your repository.
On Windows, you need to use Cygwin to run Ansible commands.

```bash
# start a Cygwin bash session
cygwin bash
# navigate to the dev-alchemy directory
cd /cygdrive/c/<path-to-your-repo>/dev-alchemy/
ansible-playbook ./playbooks/setup.yml -i ./inventory/hyperv_ubuntu.yml -l ubuntu_host -vvv --check
ansible-playbook ./playbooks/setup.yml -i ./inventory/hyperv_ubuntu.yml -l ubuntu_host -vvv
```

## Destroying the Vagrant Box

When you are done, you can destroy the Vagrant box and remove the box from your system:

```bash
cd deployments\vagrant\linux-ubuntu-hyperv
vagrant destroy
vagrant box remove linux-ubuntu --provider hyperv
cd ..\..\..
```
