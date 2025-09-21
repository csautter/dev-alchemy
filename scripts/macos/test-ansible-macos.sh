#!/usr/bin/env bash
set -ex

# check if tart is installed
if ! command -v tart &> /dev/null
then
    echo "tart could not be found, please install it first."
    exit 1
fi

if ! tart list | grep -q "^local.*sequoia-base.*$"; then
  echo "Cloning sequoia-base image..."
  tart clone ghcr.io/cirruslabs/macos-sequoia-base:latest sequoia-base
else
  echo "sequoia-base image already exists."
fi

# optionally disable graphics with --no-graphics
# WARNING: exposing ssh port with bridged networking and insecure ssh password is very insecure and should only be used for testing purposes
# RECOMMENDED: try without --net-bridged first
# NOTE: on some systems with strict firewall rules tart vms might not get internet access without bridged networking
tart run --net-bridged="Wi-Fi" sequoia-base &

# Retry until VM_IP is not empty
VM_IP=""
while [[ -z "$VM_IP" ]]; do
    VM_IP=$(tart ip --resolver=arp sequoia-base 2>/dev/null || echo "")
    sleep 1
done
echo "VM IP: $VM_IP"

# check ssh connectivity
# Retry SSH until successful
until sshpass -p admin ssh -o "StrictHostKeyChecking no" -o "ConnectTimeout=5" admin@$VM_IP "echo 'SSH connection successful'"; do
    echo "Waiting for SSH to become available..."
    sleep 2
done

# write to inventory file
cat <<EOF > inventory/remote.yml
all:
  hosts:
    $VM_IP:
      ansible_user: admin
EOF

ansible-playbook playbooks/setup.yml -i inventory/remote.yml -l "$VM_IP" --ask-pass --ask-become-pass -v

tart stop sequoia-base

# optionally delete the VM
# tart delete sequoia-base