#!/usr/bin/env bash
set -e
# This script determines the IP address of the Windows 11 ARM64 UTM VM on macOS.

utm_vm_dir="/Users/$(whoami)/Library/Containers/com.utmapp.UTM/Data/Documents"
plist="$utm_vm_dir/Windows11-ARM64-dev-alchemy.utm/config.plist"

# Read the MAC address from the plist file
mac=$(cat "$plist" | grep -A1 'MacAddress' | grep string | awk -F'[<>]' '{print $3}')

echo "Looking for IP address associated with MAC address: $mac"

# Find the IP address associated with the MAC address
ip=$(arp -a | grep -i "$mac" | grep ifscope | awk '{print $12}' | tr -d '()')

if [ -z "$ip" ]; then
	echo "Could not find the IP address for MAC address $mac. Is the VM running?"
	exit 1
else
	echo "The IP address of the Windows 11 ARM64 UTM VM is: $ip"
fi
