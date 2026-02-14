$ErrorActionPreference = "Stop"

# write a file that logs if this file was executed
New-Item -Path "C:\" -Name "execution.log" -ItemType "file" -Force
Add-Content -Path "C:\execution.log" -Value "cloud-init-combined.ps1 executed on $(Get-Date)"

# Install Hyper-V
if ((Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V).State -ne 'Enabled') {
    Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All -NoRestart
    Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V-Management-PowerShell -All -NoRestart
    Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V-Management-Clients -All -NoRestart
    
    # create default virtual switch with NAT network (required for nested virtualization)
    if (-not (Get-VMSwitch -Name "Default Switch" -ErrorAction SilentlyContinue)) {
        New-VMSwitch -Name "Default Switch" -SwitchType Internal
        $netAdapter = Get-NetAdapter | Where-Object { $_.Name -like "vEthernet (Default Switch)" }
        if ($netAdapter) {
            New-NetIPAddress -IPAddress "192.168.0.1" -PrefixLength 24 -InterfaceIndex $netAdapter.ifIndex
            New-NetNat -Name "DefaultSwitchNAT" -InternalIPInterfaceAddressPrefix "192.168.0.0/24"
        }
    }

    # After creating the NAT, add DHCP Server
    if (-not (Get-WindowsFeature -Name DHCP -ErrorAction SilentlyContinue).Installed) {
        Install-WindowsFeature -Name DHCP -IncludeManagementTools
    }

    # Configure DHCP scope
    $scopeName = "DefaultSwitchDHCP"
    if (-not (Get-DhcpServerv4Scope -ScopeId 192.168.0.0 -ErrorAction SilentlyContinue)) {
        Add-DhcpServerv4Scope -Name $scopeName `
            -StartRange 192.168.0.100 `
            -EndRange 192.168.0.200 `
            -SubnetMask 255.255.255.0 `
            -State Active
        
        Set-DhcpServerv4OptionValue -ScopeId 192.168.0.0 `
            -Router 192.168.0.1 `
            -DnsServer 8.8.8.8,8.8.4.4
    }

    # Bind DHCP server only to the Default Switch adapter
    $switchAdapter = Get-NetAdapter | Where-Object { $_.Name -like "vEthernet (Default Switch)" }
    if ($switchAdapter) {
        Set-DhcpServerv4Binding -InterfaceAlias $switchAdapter.Name -BindingState $true
        
        # Unbind from all other interfaces to avoid conflicts with Azure VNet DHCP
        Get-DhcpServerv4Binding | Where-Object { $_.InterfaceAlias -ne $switchAdapter.Name } | 
            ForEach-Object { Set-DhcpServerv4Binding -InterfaceAlias $_.InterfaceAlias -BindingState $false }
    }

    # Authorize DHCP server (required for domain-joined machines)
    Add-DhcpServerInDC -DnsName $env:COMPUTERNAME

    # Restart to complete Hyper-V installation
    Write-Host 'Restart required to complete Hyper-V installation.'
    Restart-Computer -Force
    exit
}

$RunnerToken = "__RUNNER_TOKEN__"
$RepoUrl     = "__REPO_URL__"
$RunnerName  = $env:COMPUTERNAME
$RunnerDir   = "C:\actions-runner"

New-Item -ItemType Directory -Force -Path $RunnerDir
Set-Location $RunnerDir

# Create a local user 'ghrunner' with a random password, add to Administrators and Hyper-V Administrators


Add-Type -AssemblyName System.Web
$Password = -join ((48..57) + (65..90) + (97..122) | Get-Random -Count 20 | % {[char]$_})
Write-Host "[DEBUG] Generated password: $Password"
$SecurePassword = ConvertTo-SecureString $Password -AsPlainText -Force

# Create the user if it doesn't exist
if (-not (Get-LocalUser -Name "ghrunner" -ErrorAction SilentlyContinue)) {
    New-LocalUser -Name "ghrunner" -Password $SecurePassword -FullName "GitHub Runner" -Description "Local user for GitHub Actions Runner" -PasswordNeverExpires
}

# Add user to Administrators and Hyper-V Administrators groups
Add-LocalGroupMember -Group "Administrators" -Member "ghrunner" -ErrorAction SilentlyContinue
Add-LocalGroupMember -Group "Hyper-V Administrators" -Member "ghrunner" -ErrorAction SilentlyContinue

.\config.cmd `
  --url $RepoUrl `
  --token $RunnerToken `
  --name $RunnerName `
  --labels windows,azure,nested,$RunnerName `
  --unattended `
  --ephemeral `
  --runasservice `
  --windowslogonaccount ghrunner `
  --windowslogonpassword $Password
