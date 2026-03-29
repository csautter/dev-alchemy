$ErrorActionPreference = "Stop"

$VirtualizationFlavor = "__VIRTUALIZATION_FLAVOR__"
$LogDir = "C:\ProgramData\dev-alchemy\logs"
$SecretsDir = "C:\ProgramData\dev-alchemy\secrets"
$RunnerSetupLog = Join-Path $LogDir "runner-setup.log"
$RunnerPasswordFile = Join-Path $SecretsDir "ghrunner-password.txt"

New-Item -ItemType Directory -Path $LogDir -Force | Out-Null
New-Item -ItemType Directory -Path $SecretsDir -Force | Out-Null

function Write-SetupLog {
    param([string]$Message)

    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $line = "[$timestamp] $Message"
    Write-Host $line
    Add-Content -Path $RunnerSetupLog -Value $line
}

function Protect-SecretFile {
    param([string]$Path)

    try {
        $acl = New-Object System.Security.AccessControl.FileSecurity
        $acl.SetOwner([System.Security.Principal.NTAccount]"Administrators")
        $acl.SetAccessRuleProtection($true, $false)

        $inheritanceFlags = [System.Security.AccessControl.InheritanceFlags]::None
        $propagationFlags = [System.Security.AccessControl.PropagationFlags]::None
        $allow = [System.Security.AccessControl.AccessControlType]::Allow

        $administratorsRule = New-Object System.Security.AccessControl.FileSystemAccessRule("Administrators", "FullControl", $inheritanceFlags, $propagationFlags, $allow)
        $systemRule = New-Object System.Security.AccessControl.FileSystemAccessRule("SYSTEM", "FullControl", $inheritanceFlags, $propagationFlags, $allow)

        $null = $acl.AddAccessRule($administratorsRule)
        $null = $acl.AddAccessRule($systemRule)

        Set-Acl -Path $Path -AclObject $acl
    } catch {
        Write-SetupLog "Warning: Failed to tighten ACLs on secret file '$Path': $($_.Exception.Message)"
    }
}

function Write-HyperVNetworkSnapshot {
    param([string]$Label)

    Write-SetupLog "Hyper-V network snapshot: $Label"

    try {
        $switches = Get-VMSwitch -ErrorAction SilentlyContinue | Format-List * | Out-String -Width 4096
        Add-Content -Path $RunnerSetupLog -Value $switches
    } catch {
        Write-SetupLog "Warning: Failed to list VMSwitches: $($_.Exception.Message)"
    }

    try {
        $adapters = Get-NetAdapter | Sort-Object Name | Format-List Name, InterfaceDescription, Status, MacAddress, LinkSpeed, ifIndex | Out-String -Width 4096
        Add-Content -Path $RunnerSetupLog -Value $adapters
    } catch {
        Write-SetupLog "Warning: Failed to list NetAdapters: $($_.Exception.Message)"
    }

    try {
        $addresses = Get-NetIPAddress | Sort-Object InterfaceAlias, AddressFamily, IPAddress | Format-Table -AutoSize InterfaceAlias, IPAddress, PrefixLength, AddressFamily, PrefixOrigin, SuffixOrigin | Out-String -Width 4096
        Add-Content -Path $RunnerSetupLog -Value $addresses
    } catch {
        Write-SetupLog "Warning: Failed to list NetIPAddresses: $($_.Exception.Message)"
    }

    try {
        $nat = Get-NetNat -ErrorAction SilentlyContinue | Format-List * | Out-String -Width 4096
        if ($nat) {
            Add-Content -Path $RunnerSetupLog -Value $nat
        }
    } catch {
        Write-SetupLog "Warning: Failed to list NetNat entries: $($_.Exception.Message)"
    }
}

Write-SetupLog "Selected virtualization flavor: $VirtualizationFlavor"

# write a file that logs if this file was executed
if (-not (Test-Path "C:\execution.log")) {
    New-Item -Path "C:\" -Name "execution.log" -ItemType "file" -Force
}
Add-Content -Path "C:\execution.log" -Value "runner-setup.ps1 executed on $(Get-Date)"

# Detect OS type
$OSCaption = (Get-CimInstance Win32_OperatingSystem).Caption
$IsServer = $OSCaption -like "*Server*"
Write-SetupLog "Detected OS: $OSCaption (Server: $IsServer)"

if($VirtualizationFlavor -eq "hyperv") {
    Write-SetupLog "Proceeding with Hyper-V setup..."

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
        Write-SetupLog "Scheduled task '$taskName' created to continue after restart"
    }

    # Function to remove continuation task
    function Remove-ScriptContinuation {
        $taskName = "ContinueRunnerSetup"
        Unregister-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue -Confirm:$false
        Write-SetupLog "Scheduled task '$taskName' removed"
    }

    # Install Hyper-V based on OS type
    $needsRestart = $false

    if ($IsServer) {
        # Windows Server - use Install-WindowsFeature
        Write-SetupLog "Installing Hyper-V on Windows Server..."
        
        $hypervFeature = Get-WindowsFeature -Name Hyper-V
        if (-not $hypervFeature.Installed) {
            Write-SetupLog "Installing Hyper-V role..."
            $result = Install-WindowsFeature -Name Hyper-V -IncludeManagementTools
            if ($result.RestartNeeded -eq 'Yes') {
                $needsRestart = $true
            }
        }
        
        # Ensure PowerShell module is installed
        $hypervPowerShell = Get-WindowsFeature -Name Hyper-V-PowerShell
        if (-not $hypervPowerShell.Installed) {
            Write-SetupLog "Installing Hyper-V PowerShell module..."
            $result = Install-WindowsFeature -Name Hyper-V-PowerShell
            if ($result.RestartNeeded -eq 'Yes') {
                $needsRestart = $true
            }
        }
    } else {
        # Windows Client - use Enable-WindowsOptionalFeature
        Write-SetupLog "Installing Hyper-V on Windows Client..."
        
        $hypervFeature = Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V
        if ($hypervFeature.State -ne 'Enabled') {
            Write-SetupLog "Enabling Hyper-V feature..."
            Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V-All -All -NoRestart | Out-Null
            $needsRestart = $true
        }
    }

    # Handle restart if needed
    if ($needsRestart) {
        Write-SetupLog "Restart required to complete Hyper-V installation."
        
        # Set up continuation after restart
        $scriptPath = $MyInvocation.MyCommand.Path
        if (-not $scriptPath) {
            $scriptPath = "C:\runner-setup.ps1"  # Fallback path
        }
        Set-ScriptContinuation -ScriptPath $scriptPath
        
        Write-SetupLog "System will restart in 10 seconds. Script will continue automatically after restart."
        Start-Sleep -Seconds 10
        Restart-Computer -Force
        exit
    }

    # Remove continuation task since we're past the restart phase
    Remove-ScriptContinuation

    # Import Hyper-V module
    Write-SetupLog "Importing Hyper-V module..."
    Import-Module Hyper-V -ErrorAction Stop
    Write-HyperVNetworkSnapshot -Label "before-switch-setup"

    # Create default virtual switch with NAT network (required for nested virtualization)
    $existingSwitch = Get-VMSwitch -Name "Default Switch" -ErrorAction SilentlyContinue
    if (-not $existingSwitch) {
        Write-SetupLog "Creating Default Switch..."
        try {
            New-VMSwitch -Name "Default Switch" -SwitchType Internal -ErrorAction Stop
        } catch {
            Write-SetupLog "Warning: Failed to create Default Switch. It may already exist. Error: $($_.Exception.Message)"
            # Re-check if switch now exists
            $existingSwitch = Get-VMSwitch -Name "Default Switch" -ErrorAction SilentlyContinue
        }
    }

    if ($existingSwitch -or (Get-VMSwitch -Name "Default Switch" -ErrorAction SilentlyContinue)) {
        Write-SetupLog "Default Switch exists, configuring network..."
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

    Write-HyperVNetworkSnapshot -Label "after-switch-setup"


    # Configure DHCP Server (outside the switch creation block)
    Write-SetupLog "Configuring DHCP Server..."
    if (-not (Get-WindowsFeature -Name DHCP -ErrorAction SilentlyContinue).Installed) {
        Write-SetupLog "Installing DHCP Server feature..."
        Install-WindowsFeature -Name DHCP -IncludeManagementTools
    }

    # Configure DHCP scope
    $scopeName = "DefaultSwitchDHCP"
    $existingScope = Get-DhcpServerv4Scope -ScopeId 192.168.0.0 -ErrorAction SilentlyContinue
    if (-not $existingScope) {
        Write-SetupLog "Creating DHCP scope..."
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
        Write-SetupLog "Binding DHCP to Default Switch adapter..."
        Set-DhcpServerv4Binding -InterfaceAlias $switchAdapter.Name -BindingState $true -ErrorAction SilentlyContinue
        
        # Unbind from all other interfaces to avoid conflicts with Azure VNet DHCP
        Get-DhcpServerv4Binding | Where-Object { $_.InterfaceAlias -ne $switchAdapter.Name } | 
            ForEach-Object { Set-DhcpServerv4Binding -InterfaceAlias $_.InterfaceAlias -BindingState $false -ErrorAction SilentlyContinue }
    }

    # Authorize DHCP server (only if domain-joined)
    if ((Get-WmiObject Win32_ComputerSystem).PartOfDomain) {
        Write-SetupLog "Authorizing DHCP server in Active Directory..."
        Add-DhcpServerInDC -DnsName $env:COMPUTERNAME -ErrorAction SilentlyContinue
    }

    Write-HyperVNetworkSnapshot -Label "after-dhcp-setup"
}

$RunnerToken = "__RUNNER_TOKEN__"
$RepoUrl     = "__REPO_URL__"
$RunnerName  = $env:COMPUTERNAME
$RunnerDir   = "C:\actions-runner"
$RunnerUser   = "ghrunner"

New-Item -ItemType Directory -Force -Path $RunnerDir
Set-Location $RunnerDir

# Create a local user 'ghrunner' with a random password, add to Administrators and Hyper-V Administrators


Add-Type -AssemblyName System.Web
$Password = -join ((48..57) + (65..90) + (97..122) | Get-Random -Count 20 | % {[char]$_})
$SecurePassword = ConvertTo-SecureString $Password -AsPlainText -Force
Set-Content -Path $RunnerPasswordFile -Value $Password -NoNewline -Force
Protect-SecretFile -Path $RunnerPasswordFile
Write-SetupLog "Stored runner password in protected secret file: $RunnerPasswordFile"

# Create the user if it doesn't exist
if (-not (Get-LocalUser -Name $RunnerUser -ErrorAction SilentlyContinue)) {
    New-LocalUser -Name $RunnerUser -Password $SecurePassword -FullName "GitHub Runner" -Description "Local user for GitHub Actions Runner" -PasswordNeverExpires
}

# Add user to Administrators and Hyper-V Administrators groups
Add-LocalGroupMember -Group "Administrators" -Member $RunnerUser -ErrorAction SilentlyContinue
Add-LocalGroupMember -Group "Hyper-V Administrators" -Member $RunnerUser -ErrorAction SilentlyContinue

.\config.cmd `
  --url $RepoUrl `
  --token $RunnerToken `
  --name $RunnerName `
  --labels windows,azure,nested,$RunnerName `
  --ephemeral `
  --unattended `
  --runasservice `
  --windowslogonaccount $RunnerUser `
  --windowslogonpassword $Password

# Allow the SCM a moment to register the service after config
Start-Sleep -Seconds 5

# Configure automatic service recovery on failure.
# For ephemeral runners we intentionally leave failureflag at its default (0) so that
# a clean exit after completing a job does NOT trigger a restart — only crashes /
# non-zero exits will. Restart delays: 30 s → 60 s → 120 s; reset counter after 1 h.
Write-SetupLog "Configuring GitHub Actions Runner service auto-recovery..."

# Derive the service name from the repo URL (https://github.com/<owner>/<repo>)
$urlParts   = $RepoUrl.TrimEnd('/').Split('/')
$repoOwner  = $urlParts[-2]
$repoName   = $urlParts[-1]
$ServiceName = "actions.runner.$repoOwner.$repoName.$RunnerName"

$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

# Fallback: search by wildcard in case the name differs slightly
if (-not $svc) {
    Write-Warning "Service '$ServiceName' not found; searching by pattern..."
    $svc = Get-Service | Where-Object { $_.Name -like "actions.runner*$RunnerName*" } | Select-Object -First 1
}

if ($svc) {
    Write-SetupLog "Applying recovery policy to service: $($svc.Name)"
    # reset=3600  → reset failure count after 3600 s (1 h) of stable operation
    # actions     → restart after 30 s on 1st failure, 60 s on 2nd, 120 s on all subsequent
    sc.exe failure "$($svc.Name)" reset=3600 actions=restart/30000/restart/60000/restart/120000
    Write-SetupLog "Service auto-recovery configured successfully."
} else {
    Write-Warning "Could not locate the runner service to configure recovery. Skipping."
}
