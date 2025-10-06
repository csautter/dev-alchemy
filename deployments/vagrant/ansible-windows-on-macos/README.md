# Run Ansible Windows with Vagrant and QEMU on MacOS

This guide will help you set up and run Ansible playbooks on a Windows VM using Vagrant with QEMU as the provider.
All commands are meant to be run in a bash or zsh terminal on a MacOS host machine.

## Prerequisites

Use the [dev-alchemy-self-setup.sh](../../../scripts/macos/dev-alchemy-self-setup.sh) script to set up your macos machine with the necessary tools and configurations.

Ensure you have the following installed:

- [Vagrant](https://www.vagrantup.com/downloads)
- [Ansible](https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html)
- [QEMU](https://www.qemu.org/download/)
- [libvirt](https://libvirt.org/)
- [UTM](https://mac.getutm.app/) (optional, for managing VMs)

## Adding the Vagrant Box

Patch the metadata of the Vagrant box to make it compatible with the QEMU provider:

```bash
previous_dir=$(pwd)
cd ./vendor/windows
mkdir -p ./tmpbox
tar -xvf ./win11-qemu.box -C ./tmpbox
cd ./tmpbox
mv ./metadata.json ./metadata.json.bak
cat metadata.json.bak | jq '.provider = "qemu"' > ./metadata.json
tar -cvf ../win11-qemu.box .
cd ..
rm -rf ./tmpbox
cd $previous_dir
```

Load the Vagrant box and start the VM using QEMU as the provider:

```bash
vagrant box add win11-packer-qemu ./vendor/windows/win11-qemu.box --provider qemu
cd ./deployments/vagrant/ansible-windows-on-macos
vagrant up --provider qemu
```

After the VM is up, you can connect to it using UTM. The default credentials are:

- Username: `Administrator`
- Password: `P@ssw0rd!`

## Determine the IP Address of the Vagrant Box

You can find the IP address of the Vagrant box using the following command:

```bash
vagrant winrm -c "ipconfig"
```

Alternatively, you can use PowerShell to extract the IP address directly:

```bash
vagrant_ip=$(vagrant winrm -c "ipconfig" | grep -Eo 'IPv4 Address[^\:]*: ([0-9]{1,3}\.){3}[0-9]{1,3}' | awk -F': ' '{print $2}')
echo "Vagrant Box IP Address: $vagrant_ip"
```

## Write the Inventory File

You can create the inventory file using Bash and the `$vagrant_ip` variable:

```bash
cat > ./inventory/qemu_windows_winrm.yml <<EOF
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
EOF
echo "Inventory file created at ./inventory/qemu_windows_winrm.yml"
```
```

## Run Ansible Playbook

Run the Ansible playbook using the created inventory file. Make sure to replace `<path-to-your-repo>` with the actual path to your repository.

```bash
# start a Cygwin bash session
cygwin bash
# navigate to the dev-alchemy directory
cd /cygdrive/c/<path-to-your-repo>/dev-alchemy/
ansible-playbook ./playbooks/setup.yml -i ./inventory/qemu_windows_winrm.yml -l windows_host -vvv --check
ansible-playbook ./playbooks/setup.yml -i ./inventory/qemu_windows_winrm.yml -l windows_host -vvv
```

## Destroying the Vagrant Box

When you are done, you can destroy the Vagrant box and remove the box from your system:

```bash
vagrant destroy
vagrant box remove win11-packer --provider libvirt
```
