# Adding the Vagrant Box

```bash
vagrant box add win11-packer ..\..\..\vendor\windows\win11-hyperv.box --provider hyperv
vagrant init win11-packer
vagrant up --provider hyperv
```

# Determine the IP Address of the Vagrant Box

```bash
vagrant winrm -c "ipconfig"
```

```powershell
$vagrant_ip = (vagrant winrm -c "ipconfig" | Select-String -Pattern 'IPv4 Address.*: (\d{1,3}\.){3}\d{1,3}' | ForEach-Object { $_.Matches[0].Value.Split(':')[1].Trim() })
Write-Output "Vagrant Box IP Address: $vagrant_ip"$
```

# Write the Inventory File

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
$inventory | Set-Content -Path "../../../inventory/hyperv_windows_winrm.yml"
Write-Output "Inventory file created at ../../../inventory/hyperv_windows_winrm.yml"
```

# Run Ansible Playbook

```bash
cygwin bash
cd /cygdrive/c/Users/<your-user>/<your-path>/dev-alchemy/
ansible-playbook ./playbooks/setup.yml -i ./inventory/hyperv_windows_winrm.yml -l windows_host -vvv --check
ansible-playbook ./playbooks/setup.yml -i ./inventory/hyperv_windows_winrm.yml -l windows_host -vvv
```

# Destroying the Vagrant Box

```bash
vagrant destroy
vagrant box remove win11-packer --provider hyperv
```
