$ErrorActionPreference = 'Stop'

$statePath = $env:DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH
$userName = $env:DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_USER
$passwordPlain = $env:DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_PASSWORD
$publicKey = $env:DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_SSH_PUBLIC_KEY
$forceSSHUninstall = [System.Convert]::ToBoolean($env:DEV_ALCHEMY_LOCAL_WINDOWS_FORCE_SSH_UNINSTALL)

$openSSHCapabilityName = 'OpenSSH.Server~~~~0.0.1.0'
$openSSHInstallStatusIntervalSeconds = 15
$openSSHBuiltInFirewallRuleName = 'OpenSSH-Server-In-TCP'
$localFirewallRuleName = 'DevAlchemyLocalSSHDLoopback'
$defaultShellRegistryPath = 'HKLM:\SOFTWARE\OpenSSH'
$defaultShellRegistryName = 'DefaultShell'
$defaultShellPath = 'C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe'
$administratorsAuthorizedKeysPath = Join-Path $env:ProgramData 'ssh\administrators_authorized_keys'
$administratorsSid = '*S-1-5-32-544'
$systemSid = '*S-1-5-18'

if ([string]::IsNullOrWhiteSpace($statePath)) {
    throw 'DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH is required.'
}
if ([string]::IsNullOrWhiteSpace($userName)) {
    throw 'DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_USER is required.'
}
if ([string]::IsNullOrWhiteSpace($passwordPlain)) {
    throw 'DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_PASSWORD is required.'
}
if ([string]::IsNullOrWhiteSpace($publicKey)) {
    throw 'DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_SSH_PUBLIC_KEY is required.'
}

Write-Output 'Validating local Windows SSH provision bootstrap inputs.'

function Save-State($state) {
    $state | ConvertTo-Json -Compress | Set-Content -Path $statePath -Encoding Ascii
}

function Get-WindowsCapabilityState([string]$name) {
    $capability = Get-WindowsCapability -Online -Name $name -ErrorAction Stop | Select-Object -First 1
    if ($null -eq $capability) {
        throw ('Windows capability not found: ' + $name)
    }

    return @{
        Name = [string]$capability.Name
        State = [string]$capability.State
        Installed = ([string]$capability.State -eq 'Installed')
        Pending = ([string]$capability.State -like '*Pending')
    }
}

function Test-OpenSSHCapabilityInstallNeeded($capabilityState, $sshdServiceState) {
    if ([bool]$capabilityState.Installed) {
        return $false
    }
    if ([bool]$sshdServiceState.Exists) {
        return $false
    }

    return $true
}

function Assert-OpenSSHCapabilityStateIsUsable($capabilityState, $sshdServiceState) {
    if ([bool]$capabilityState.Pending -and -not [bool]$sshdServiceState.Exists) {
        throw ('OpenSSH Server capability is in state "' + [string]$capabilityState.State + '" and sshd is unavailable. Reboot Windows to finish the pending capability change, then retry provisioning.')
    }
}

function Write-OpenSSHPendingStateGuidance($capabilityState, $sshdServiceState) {
    if (-not [bool]$capabilityState.Pending) {
        return
    }

    if ([bool]$sshdServiceState.Exists) {
        Write-Output ('Windows reports the OpenSSH Server capability as "' + [string]$capabilityState.State + '" while sshd still exists. The current sshd installation will be reused for this run, but you should reboot Windows soon to finish the pending capability change and return the OS to a consistent state.')
        return
    }

    Write-Output ('Windows reports the OpenSSH Server capability as "' + [string]$capabilityState.State + '" and sshd is unavailable. A Windows reboot is required before SSH provisioning can continue.')
}

function Install-WindowsCapabilityWithHeartbeat([string]$name, [int]$statusIntervalSeconds) {
    $powerShell = [powershell]::Create()
    $asyncResult = $null
    try {
        $null = $powerShell.AddScript({
            param($capabilityName)
            $ErrorActionPreference = 'Stop'
            Add-WindowsCapability -Online -Name $capabilityName | Out-Null
        }).AddArgument($name)

        $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
        $asyncResult = $powerShell.BeginInvoke()

        while (-not $asyncResult.AsyncWaitHandle.WaitOne([TimeSpan]::FromSeconds($statusIntervalSeconds))) {
            $elapsedSeconds = [int][Math]::Floor($stopwatch.Elapsed.TotalSeconds)
            $currentCapabilityState = Get-WindowsCapabilityState $name
            Write-Output ('OpenSSH Server capability install is still running after ' + $elapsedSeconds + ' seconds (Windows currently reports state "' + [string]$currentCapabilityState.State + '").')
        }

        $powerShell.EndInvoke($asyncResult) | Out-Null
        if ($powerShell.HadErrors) {
            $errorMessages = @(
                $powerShell.Streams.Error |
                    ForEach-Object { [string]$_ } |
                    Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
            )
            if ($errorMessages.Count -gt 0) {
                throw ('Add-WindowsCapability reported errors: ' + ($errorMessages -join ' | '))
            }

            throw 'Add-WindowsCapability reported one or more PowerShell errors.'
        }
    } finally {
        if ($null -ne $asyncResult) {
            $asyncResult.AsyncWaitHandle.Dispose()
        }
        $powerShell.Dispose()
    }
}

function Get-ServiceState([string]$name) {
    $service = Get-CimInstance -ClassName Win32_Service -Filter ("Name='" + $name + "'") -ErrorAction SilentlyContinue
    if ($null -eq $service) {
        return @{
            Exists = $false
            WasRunning = $false
            StartMode = ''
        }
    }

    return @{
        Exists = $true
        WasRunning = ([string]$service.State -eq 'Running')
        StartMode = [string]$service.StartMode
    }
}

function Get-NetFirewallRuleState([string]$name) {
    $rule = Get-NetFirewallRule -Name $name -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -eq $rule) {
        return @{
            Exists = $false
            Enabled = $false
        }
    }

    return @{
        Exists = $true
        Enabled = ($rule.Enabled -eq 'True')
    }
}

function Get-RegistryStringState([string]$path, [string]$name) {
    if (-not (Test-Path -Path $path)) {
        return @{
            Exists = $false
            Value = ''
        }
    }

    $property = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $property) {
        return @{
            Exists = $false
            Value = ''
        }
    }

    return @{
        Exists = $true
        Value = [string]$property.$name
    }
}

function Get-FileState([string]$path) {
    if (-not (Test-Path -Path $path)) {
        return @{
            Exists = $false
            ContentBase64 = ''
            Sddl = ''
        }
    }

    $bytes = [System.IO.File]::ReadAllBytes($path)
    return @{
        Exists = $true
        ContentBase64 = [System.Convert]::ToBase64String($bytes)
        Sddl = (Get-Acl -Path $path).Sddl
    }
}

function Get-LocalUserState([string]$name) {
    $localUser = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $localUser) {
        return @{
            Exists = $false
            Enabled = $false
            Description = ''
        }
    }

    return @{
        Exists = $true
        Enabled = [bool]$localUser.Enabled
        Description = [string]$localUser.Description
    }
}

function Get-LocalAdministratorsGroupName() {
    $group = Get-LocalGroup | Where-Object { $null -ne $_.SID -and $_.SID.Value -eq 'S-1-5-32-544' } | Select-Object -First 1
    if ($null -eq $group) {
        throw 'The built-in local Administrators group was not found.'
    }

    return [string]$group.Name
}

function Test-IsLocalAdministrator([string]$groupName, [string]$name) {
    return @(
        Get-LocalGroupMember -Group $groupName -ErrorAction SilentlyContinue |
            Where-Object { $_.Name -eq $name -or $_.Name -eq ('.\' + $name) -or $_.Name -match ('\\' + [regex]::Escape($name) + '$') }
    ).Count -gt 0
}

function Set-AdminAuthorizedKeysPermissions([string]$path) {
    Write-Output ('Setting OpenSSH administrators_authorized_keys ACLs with well-known SIDs on ' + $path)
    $icaclsOutput = & icacls.exe $path /inheritance:r /grant ($administratorsSid + ':F') /grant ($systemSid + ':F') 2>&1
    $exitCode = $LASTEXITCODE
    if ($null -ne $icaclsOutput) {
        $icaclsOutput | ForEach-Object { Write-Output ('icacls: ' + $_) }
    }
    if ($exitCode -ne 0) {
        throw ('icacls.exe failed while securing administrators_authorized_keys with exit code ' + $exitCode)
    }
}

function Write-StateSummary($state) {
    Write-Output ('OpenSSH capability state: installed=' + [string][bool]$state.CapabilityInstalled + ', state=' + [string]$state.CapabilityState)
    Write-Output ('sshd service state: existed=' + [string][bool]$state.SshdServiceExisted + ', running=' + [string][bool]$state.SshdServiceWasRunning + ', startMode=' + [string]$state.SshdServiceStartMode)
    Write-Output ('Firewall state: builtInRule existed=' + [string][bool]$state.BuiltInFirewallRuleExisted + ', enabled=' + [string][bool]$state.BuiltInFirewallRuleEnabled + '; loopbackRule existed=' + [string][bool]$state.LocalFirewallRuleExisted + ', enabled=' + [string][bool]$state.LocalFirewallRuleEnabled)
    Write-Output ('OpenSSH shell state: defaultShell existed=' + [string][bool]$state.DefaultShellExisted + ', value=' + [string]$state.DefaultShellValue)
    Write-Output ('Authorized keys state: existed=' + [string][bool]$state.AuthorizedKeysExisted + ', path=' + $administratorsAuthorizedKeysPath)
    Write-Output ('Local user state: existed=' + [string][bool]$state.UserExisted + ', enabled=' + [string][bool]$state.UserWasEnabled + ', wasAdministrator=' + [string][bool]$state.UserWasAdministrator + ', description=' + [string]$state.UserDescription)
    Write-Output ('Force SSH uninstall requested: ' + [string][bool]$state.ForceSSHUninstall)
}

$capabilityState = Get-WindowsCapabilityState $openSSHCapabilityName
$sshdServiceState = Get-ServiceState 'sshd'
$builtInFirewallRuleState = Get-NetFirewallRuleState $openSSHBuiltInFirewallRuleName
$localFirewallRuleState = Get-NetFirewallRuleState $localFirewallRuleName
$defaultShellState = Get-RegistryStringState $defaultShellRegistryPath $defaultShellRegistryName
$authorizedKeysState = Get-FileState $administratorsAuthorizedKeysPath
$userState = Get-LocalUserState $userName
$administratorsGroupName = Get-LocalAdministratorsGroupName
$userWasAdministrator = $false
if ($userState.Exists) {
    $userWasAdministrator = Test-IsLocalAdministrator $administratorsGroupName $userName
}

$state = @{
    CapabilityInstalled = [bool]$capabilityState.Installed
    CapabilityState = [string]$capabilityState.State
    CapabilityPending = [bool]$capabilityState.Pending
    CapabilityInstallManaged = $false
    SshdServiceExisted = [bool]$sshdServiceState.Exists
    SshdServiceWasRunning = [bool]$sshdServiceState.WasRunning
    SshdServiceStartMode = [string]$sshdServiceState.StartMode
    BuiltInFirewallRuleExisted = [bool]$builtInFirewallRuleState.Exists
    BuiltInFirewallRuleEnabled = [bool]$builtInFirewallRuleState.Enabled
    LocalFirewallRuleExisted = [bool]$localFirewallRuleState.Exists
    LocalFirewallRuleEnabled = [bool]$localFirewallRuleState.Enabled
    DefaultShellExisted = [bool]$defaultShellState.Exists
    DefaultShellValue = [string]$defaultShellState.Value
    AuthorizedKeysExisted = [bool]$authorizedKeysState.Exists
    AuthorizedKeysContentBase64 = [string]$authorizedKeysState.ContentBase64
    AuthorizedKeysSddl = [string]$authorizedKeysState.Sddl
    UserName = $userName
    UserExisted = [bool]$userState.Exists
    UserWasEnabled = [bool]$userState.Enabled
    UserDescription = [string]$userState.Description
    UserWasAdministrator = [bool]$userWasAdministrator
    ForceSSHUninstall = [bool]$forceSSHUninstall
}
Save-State $state
Write-Output 'Captured existing OpenSSH, firewall, shell, key, and local user state.'
Write-StateSummary $state
Write-OpenSSHPendingStateGuidance $capabilityState $sshdServiceState
Assert-OpenSSHCapabilityStateIsUsable $capabilityState $sshdServiceState

if (Test-OpenSSHCapabilityInstallNeeded $capabilityState $sshdServiceState) {
    Write-Output ('Installing the OpenSSH Server capability. Progress updates will be logged every ' + $openSSHInstallStatusIntervalSeconds + ' seconds while Windows completes the capability change.')
    Install-WindowsCapabilityWithHeartbeat $openSSHCapabilityName $openSSHInstallStatusIntervalSeconds
    $state.CapabilityInstallManaged = $true
    Save-State $state
    $postInstallCapabilityState = Get-WindowsCapabilityState $openSSHCapabilityName
    Write-Output ('OpenSSH Server capability install completed for this provisioning run with state "' + [string]$postInstallCapabilityState.State + '".')
} elseif ([bool]$state.CapabilityInstalled) {
    Write-Output ('Reusing the existing OpenSSH Server capability in state "' + [string]$state.CapabilityState + '".')
} elseif ([bool]$state.SshdServiceExisted) {
    Write-Output ('OpenSSH capability is reported as "' + [string]$state.CapabilityState + '" but sshd already exists, so the current installation will be reused without reinstalling the capability.')
} else {
    Write-Output ('OpenSSH capability is reported as "' + [string]$state.CapabilityState + '" and no install action was required.')
}

Write-Output 'Preparing the sshd service for loopback-only provisioning.'
$sshdService = Get-Service -Name 'sshd' -ErrorAction SilentlyContinue
if ($null -eq $sshdService) {
    throw 'The sshd service was not found after enabling OpenSSH Server.'
}
if (([string]$state.SshdServiceStartMode).ToLowerInvariant() -eq 'disabled') {
    Set-Service -Name 'sshd' -StartupType Manual
}

if (-not [bool]$state.BuiltInFirewallRuleExisted) {
    $builtInRule = Get-NetFirewallRule -Name $openSSHBuiltInFirewallRuleName -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -ne $builtInRule) {
        Write-Output 'Disabling the broad OpenSSH firewall rule created during install.'
        $builtInRule | Disable-NetFirewallRule | Out-Null
    }
}

if (-not [bool]$state.LocalFirewallRuleExisted) {
    Write-Output 'Creating the loopback-only OpenSSH firewall rule.'
    New-NetFirewallRule -Name $localFirewallRuleName -DisplayName 'Dev Alchemy Local OpenSSH Loopback' -Direction Inbound -Action Allow -Protocol TCP -LocalAddress 127.0.0.1 -LocalPort 22 -Profile Any | Out-Null
} elseif (-not [bool]$state.LocalFirewallRuleEnabled) {
    Write-Output 'Enabling the existing loopback-only OpenSSH firewall rule.'
    Enable-NetFirewallRule -Name $localFirewallRuleName | Out-Null
} else {
    Write-Output 'Keeping the existing loopback-only OpenSSH firewall rule enabled.'
}

Write-Output 'Setting the OpenSSH default shell to PowerShell for Ansible.'
if (-not (Test-Path -Path $defaultShellRegistryPath)) {
    New-Item -Path $defaultShellRegistryPath -Force | Out-Null
}
New-ItemProperty -Path $defaultShellRegistryPath -Name $defaultShellRegistryName -Value $defaultShellPath -PropertyType String -Force | Out-Null

Write-Output 'Creating or updating the temporary local Ansible account.'
$securePassword = ConvertTo-SecureString -String $passwordPlain -AsPlainText -Force
$localUser = Get-LocalUser -Name $userName -ErrorAction SilentlyContinue
if ($null -eq $localUser) {
    Write-Output ('Creating the temporary local user "' + $userName + '".')
    $localUser = New-LocalUser -Name $userName -Password $securePassword -PasswordNeverExpires -Description 'Dev Alchemy Ansible acct'
} else {
    Write-Output ('Reusing the existing local user "' + $userName + '" (enabled=' + [string][bool]$localUser.Enabled + ', description=' + [string]$localUser.Description + ').')
    Set-LocalUser -Name $userName -Password $securePassword -Description 'Dev Alchemy Ansible acct'
    Enable-LocalUser -Name $userName
    $localUser = Get-LocalUser -Name $userName
}

Write-Output 'Ensuring the temporary local Ansible account is an administrator.'
if (-not (Test-IsLocalAdministrator $administratorsGroupName $userName)) {
    Add-LocalGroupMember -Group $administratorsGroupName -Member $userName
    Write-Output ('Added "' + $userName + '" to the local Administrators group "' + $administratorsGroupName + '".')
} else {
    Write-Output ('The local user "' + $userName + '" is already a member of "' + $administratorsGroupName + '".')
}

Write-Output 'Installing the temporary SSH public key for administrator logins.'
$sshDirectory = Split-Path -Parent $administratorsAuthorizedKeysPath
if (-not (Test-Path -Path $sshDirectory)) {
    Write-Output ('Creating the SSH directory "' + $sshDirectory + '".')
    New-Item -Path $sshDirectory -ItemType Directory -Force | Out-Null
}
$existingAuthorizedKeys = ''
if (Test-Path -Path $administratorsAuthorizedKeysPath) {
    Write-Output ('Reading the existing administrators_authorized_keys file from "' + $administratorsAuthorizedKeysPath + '".')
    $existingAuthorizedKeys = [System.IO.File]::ReadAllText($administratorsAuthorizedKeysPath)
}
$normalizedAuthorizedKeys = $existingAuthorizedKeys.TrimEnd("`r", "`n")
if (-not [string]::IsNullOrWhiteSpace($normalizedAuthorizedKeys)) {
    $normalizedAuthorizedKeys += "`r`n"
}
$normalizedAuthorizedKeys += ($publicKey.Trim() + "`r`n")
[System.IO.File]::WriteAllText($administratorsAuthorizedKeysPath, $normalizedAuthorizedKeys, [System.Text.UTF8Encoding]::new($false))
Set-AdminAuthorizedKeysPermissions $administratorsAuthorizedKeysPath

Write-Output 'Starting the sshd service.'
Start-Service -Name 'sshd'

Write-Output 'Local Windows SSH provision bootstrap completed.'
