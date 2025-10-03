# PowerShell script to package a Hyper-V Vagrant .box file from Packer output

param(
    [string]$VhdxPath = ".\Virtual Hard Disks\win11-packer.vhdx",
    [string]$BoxName = "win11-packer.box"
)

$oldLocation = Get-Location
Set-Location -Path "..\..\..\vendor\windows\hyperv"

# Ensure the VHDX exists
if (!(Test-Path $VhdxPath)) {
    Write-Error "VHDX file not found: $VhdxPath"
    exit 1
}

# Create metadata.json
$metadata = @{
    provider = "hyperv"
    format = "vhdx"
}
$metadata | ConvertTo-Json | Set-Content -Encoding UTF8 metadata.json
Get-Content metadata.json

# Create Vagrantfile
$vagrantfileContent = @"
Vagrant.configure("2") do |config|
    config.vm.box = "win11-packer"
    config.vm.provider "hyperv" do |h|
        h.memory = 4096
        h.cpus = 2
        h.enable_virtualization_extensions = true
    end
end
"@
$vagrantfileContent | Set-Content -Encoding UTF8 Vagrantfile
Get-Content Vagrantfile

# Copy VHDX to current directory with a standard name
$boxVhdx = "box.vhdx"
Copy-Item $VhdxPath $boxVhdx -Force

# Remove old box if exists
if (Test-Path $BoxName) { Remove-Item $BoxName -Force }

# Create tar archive with 7-Zip
& "C:\Program Files\7-Zip\7z.exe" a -tzip -mx=1 $BoxName metadata.json $boxVhdx

# Clean up
Remove-Item metadata.json
Remove-Item $boxVhdx

Write-Host "Vagrant box created: $BoxName"

Set-Location -Path $oldLocation