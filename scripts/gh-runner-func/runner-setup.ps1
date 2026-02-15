$ErrorActionPreference = "Stop"

# write a file that logs if this file was executed
if (-not (Test-Path "C:\execution.log")) {
    New-Item -Path "C:\" -Name "execution.log" -ItemType "file" -Force
}
Add-Content -Path "C:\execution.log" -Value "runner-setup.ps1 executed on $(Get-Date)"

# Detect OS type
$OSCaption = (Get-CimInstance Win32_OperatingSystem).Caption
$IsServer = $OSCaption -like "*Server*"
Write-Host "Detected OS: $OSCaption (Server: $IsServer)"

# Function to ensure script re-runs after restart
function Set-ScriptContinuation {
    param([string]$ScriptPath)
    
    $taskName = "ContinueRunnerSetup"
    $action = New-ScheduledTaskAction -Execute "PowerShell.exe" -Argument "-NoProfile -ExecutionPolicy Bypass -File `"$ScriptPath`""
    $trigger = New-ScheduledTaskTrigger -AtStartup
    $principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    
    # Remove existing task if present
    Unregister-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue -Confirm:$false
    
    # Register new task
    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings | Out-Null
    Write-Host "Scheduled task '$taskName' created to continue after restart"
}

# Function to remove continuation task
function Remove-ScriptContinuation {
    $taskName = "ContinueRunnerSetup"
    Unregister-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue -Confirm:$false
    Write-Host "Scheduled task '$taskName' removed"
}

# Install Hyper-V based on OS type
$needsRestart = $false

if ($IsServer) {
    # Windows Server - use Install-WindowsFeature
    Write-Host "Installing Hyper-V on Windows Server..."
    
    $hypervFeature = Get-WindowsFeature -Name Hyper-V
    if (-not $hypervFeature.Installed) {
        Write-Host "Installing Hyper-V role..."
        $result = Install-WindowsFeature -Name Hyper-V -IncludeManagementTools
        if ($result.RestartNeeded -eq 'Yes') {
            $needsRestart = $true
        }
    }
    
    # Ensure PowerShell module is installed
    $hypervPowerShell = Get-WindowsFeature -Name Hyper-V-PowerShell
    if (-not $hypervPowerShell.Installed) {
        Write-Host "Installing Hyper-V PowerShell module..."
        $result = Install-WindowsFeature -Name Hyper-V-PowerShell
        if ($result.RestartNeeded -eq 'Yes') {
            $needsRestart = $true
        }
    }
} else {
    # Windows Client - use Enable-WindowsOptionalFeature
    Write-Host "Installing Hyper-V on Windows Client..."
    
    $hypervFeature = Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V
    if ($hypervFeature.State -ne 'Enabled') {
        Write-Host "Enabling Hyper-V feature..."
        Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V-All -All -NoRestart | Out-Null
        $needsRestart = $true
    }
}

# Handle restart if needed
if ($needsRestart) {
    Write-Host "Restart required to complete Hyper-V installation."
    
    # Set up continuation after restart
    $scriptPath = $MyInvocation.MyCommand.Path
    if (-not $scriptPath) {
        $scriptPath = "C:\runner-setup.ps1"  # Fallback path
    }
    Set-ScriptContinuation -ScriptPath $scriptPath
    
    Write-Host "System will restart in 10 seconds. Script will continue automatically after restart."
    Start-Sleep -Seconds 10
    Restart-Computer -Force
    exit
}

# Remove continuation task since we're past the restart phase
Remove-ScriptContinuation

# Import Hyper-V module
Write-Host "Importing Hyper-V module..."
Import-Module Hyper-V -ErrorAction Stop

# Create default virtual switch with NAT network (required for nested virtualization)
$existingSwitch = Get-VMSwitch -Name "Default Switch" -ErrorAction SilentlyContinue
if (-not $existingSwitch) {
    Write-Host "Creating Default Switch..."
    try {
        New-VMSwitch -Name "Default Switch" -SwitchType Internal -ErrorAction Stop
    } catch {
        Write-Host "Warning: Failed to create Default Switch. It may already exist. Error: $_"
        # Re-check if switch now exists
        $existingSwitch = Get-VMSwitch -Name "Default Switch" -ErrorAction SilentlyContinue
    }
}

if ($existingSwitch -or (Get-VMSwitch -Name "Default Switch" -ErrorAction SilentlyContinue)) {
    Write-Host "Default Switch exists, configuring network..."
    $netAdapter = Get-NetAdapter | Where-Object { $_.Name -like "vEthernet (Default Switch)" }
    if ($netAdapter) {
        # Check if IP is already configured
        $existingIP = Get-NetIPAddress -InterfaceIndex $netAdapter.ifIndex -IPAddress "192.168.0.1" -ErrorAction SilentlyContinue
        if (-not $existingIP) {
            New-NetIPAddress -IPAddress "192.168.0.1" -PrefixLength 24 -InterfaceIndex $netAdapter.ifIndex -ErrorAction SilentlyContinue
        }
        
        # Check if NAT already exists
        $existingNat = Get-NetNat -Name "DefaultSwitchNAT" -ErrorAction SilentlyContinue
        if (-not $existingNat) {
            New-NetNat -Name "DefaultSwitchNAT" -InternalIPInterfaceAddressPrefix "192.168.0.0/24" -ErrorAction SilentlyContinue
        }
    }
}



# Configure DHCP Server (outside the switch creation block)
Write-Host "Configuring DHCP Server..."
if (-not (Get-WindowsFeature -Name DHCP -ErrorAction SilentlyContinue).Installed) {
    Write-Host "Installing DHCP Server feature..."
    Install-WindowsFeature -Name DHCP -IncludeManagementTools
}

# Configure DHCP scope
$scopeName = "DefaultSwitchDHCP"
$existingScope = Get-DhcpServerv4Scope -ScopeId 192.168.0.0 -ErrorAction SilentlyContinue
if (-not $existingScope) {
    Write-Host "Creating DHCP scope..."
    Add-DhcpServerv4Scope -Name $scopeName `
        -StartRange 192.168.0.100 `
        -EndRange 192.168.0.200 `
        -SubnetMask 255.255.255.0 `
        -State Active -ErrorAction SilentlyContinue
    
    Set-DhcpServerv4OptionValue -ScopeId 192.168.0.0 `
        -Router 192.168.0.1 `
        -DnsServer 8.8.8.8,8.8.4.4 -ErrorAction SilentlyContinue
}

# Bind DHCP server only to the Default Switch adapter
$switchAdapter = Get-NetAdapter | Where-Object { $_.Name -like "vEthernet (Default Switch)" }
if ($switchAdapter) {
    Write-Host "Binding DHCP to Default Switch adapter..."
    Set-DhcpServerv4Binding -InterfaceAlias $switchAdapter.Name -BindingState $true -ErrorAction SilentlyContinue
    
    # Unbind from all other interfaces to avoid conflicts with Azure VNet DHCP
    Get-DhcpServerv4Binding | Where-Object { $_.InterfaceAlias -ne $switchAdapter.Name } | 
        ForEach-Object { Set-DhcpServerv4Binding -InterfaceAlias $_.InterfaceAlias -BindingState $false -ErrorAction SilentlyContinue }
}

# Authorize DHCP server (only if domain-joined)
if ((Get-WmiObject Win32_ComputerSystem).PartOfDomain) {
    Write-Host "Authorizing DHCP server in Active Directory..."
    Add-DhcpServerInDC -DnsName $env:COMPUTERNAME -ErrorAction SilentlyContinue
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
  --ephemeral `
  --unattended `
  --runasservice `
  --windowslogonaccount ghrunner `
  --windowslogonpassword $Password
