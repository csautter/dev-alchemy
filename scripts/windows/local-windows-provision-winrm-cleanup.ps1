$ErrorActionPreference = 'Stop'

$statePath = $env:DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH
$forceWinRMUninstall = [System.Convert]::ToBoolean($env:DEV_ALCHEMY_LOCAL_WINDOWS_FORCE_WINRM_UNINSTALL)
if ([string]::IsNullOrWhiteSpace($statePath) -or -not (Test-Path -Path $statePath)) {
    Write-Output 'No local Windows provision state file was found; skipping cleanup.'
    return
}

Write-Output 'Loading local Windows provision cleanup state.'
$state = Get-Content -Path $statePath -Raw | ConvertFrom-Json

function Restore-WinRMServiceState([bool]$wasRunning, [string]$startMode) {
    $service = Get-Service -Name 'WinRM' -ErrorAction SilentlyContinue
    if ($null -eq $service) {
        return
    }

    if ($wasRunning) {
        if ($service.Status -ne 'Running') {
            Start-Service -Name 'WinRM'
            $service = Get-Service -Name 'WinRM'
        }
    } elseif ($service.Status -eq 'Running') {
        Stop-Service -Name 'WinRM' -Force
        $service = Get-Service -Name 'WinRM'
    }

    switch ($startMode.ToLowerInvariant()) {
        'auto' {
            Set-Service -Name 'WinRM' -StartupType Automatic
        }
        'manual' {
            Set-Service -Name 'WinRM' -StartupType Manual
        }
        'disabled' {
            if ($service.Status -eq 'Running') {
                Stop-Service -Name 'WinRM' -Force
            }
            Set-Service -Name 'WinRM' -StartupType Disabled
        }
    }
}

function Restore-RegistryDWORDState([string]$path, [string]$name, [bool]$existed, [int]$value) {
    if ($existed) {
        if (-not (Test-Path -Path $path)) {
            New-Item -Path $path -Force | Out-Null
        }
        if ($null -eq (Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue)) {
            New-ItemProperty -Path $path -Name $name -Value $value -PropertyType DWord -Force | Out-Null
        } else {
            Set-ItemProperty -Path $path -Name $name -Value $value
        }
        return
    }

    if (Test-Path -Path $path) {
        Remove-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
    }
}

function Disable-WinRMFirewallRules() {
    $rules = Get-NetFirewallRule -DisplayGroup 'Windows Remote Management' -ErrorAction SilentlyContinue
    if ($null -ne $rules) {
        $rules | Disable-NetFirewallRule | Out-Null
    }
}

function Restore-FirewallRuleState([string]$name, [bool]$existed, [bool]$enabled) {
    $rule = Get-NetFirewallRule -Name $name -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $existed) {
        if ($null -ne $rule) {
            $rule | Remove-NetFirewallRule | Out-Null
        }
        return
    }

    if ($null -eq $rule) {
        return
    }

    if ($enabled) {
        $rule | Enable-NetFirewallRule | Out-Null
    } else {
        $rule | Disable-NetFirewallRule | Out-Null
    }
}

$userName = [string]$state.UserName
$administratorsGroupName = Get-LocalAdministratorsGroupName
if (-not [string]::IsNullOrWhiteSpace($userName)) {
    $localUserCleanupPlan = Get-ManagedLocalUserCleanupPlan ([bool]$state.UserExisted) $forceWinRMUninstall 'WinRM'
    Write-Output ([string]$localUserCleanupPlan.Message)
    if ([bool]$localUserCleanupPlan.RestoreUser) {
        Restore-ManagedLocalUserState $administratorsGroupName $state $userName
    } else {
        Remove-ManagedLocalUserIfPresent $administratorsGroupName $userName
    }
}

if ([bool]$state.CreatedHttpsListener) {
    Write-Output 'Removing the localhost WinRM HTTPS listener created for provisioning.'
    Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction SilentlyContinue |
        Where-Object { $_.Keys -match 'Transport=HTTPS' -and $_.Keys -match 'Port=5986' } |
        ForEach-Object { Remove-Item -Path $_.PSPath -Recurse -Force }
}

$originalListenerKeys = @()
if ($null -ne $state.ListenerKeys) {
    $originalListenerKeys = @($state.ListenerKeys | ForEach-Object { [string]$_ })
}
Write-Output 'Removing temporary WinRM listeners that were added during provisioning.'
Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction SilentlyContinue |
    Where-Object { $originalListenerKeys -notcontains [string]$_.Keys } |
    ForEach-Object { Remove-Item -Path $_.PSPath -Recurse -Force }

Write-Output 'Restoring WinRM authentication, registry, and firewall settings.'
if ($forceWinRMUninstall -and (Test-Path -Path 'WSMan:\localhost\Service\Auth\Basic')) {
    Set-Item -Path 'WSMan:\localhost\Service\Auth\Basic' -Value $false
} elseif ([bool]$state.BasicAuthPathExisted -and (Test-Path -Path 'WSMan:\localhost\Service\Auth\Basic')) {
    Set-Item -Path 'WSMan:\localhost\Service\Auth\Basic' -Value ([bool]$state.BasicAuthEnabled)
}
if ($forceWinRMUninstall -and (Test-Path -Path 'WSMan:\localhost\Service\AllowUnencrypted')) {
    Set-Item -Path 'WSMan:\localhost\Service\AllowUnencrypted' -Value $false
} elseif ([bool]$state.AllowUnencryptedPathExisted -and (Test-Path -Path 'WSMan:\localhost\Service\AllowUnencrypted')) {
    Set-Item -Path 'WSMan:\localhost\Service\AllowUnencrypted' -Value ([bool]$state.AllowUnencryptedEnabled)
}
Restore-RegistryDWORDState 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' 'LocalAccountTokenFilterPolicy' ([bool]$state.LocalAccountTokenFilterPolicyExisted) ([int]$state.LocalAccountTokenFilterPolicyValue)
Restore-FirewallRuleState 'DevAlchemyLocalWinRMHTTPS' ([bool]$state.FirewallRuleExisted) ([bool]$state.FirewallRuleEnabled)

$certificateThumbprint = [string]$state.CreatedCertificateThumbprint
if (-not [string]::IsNullOrWhiteSpace($certificateThumbprint)) {
    Write-Output 'Removing the temporary localhost WinRM certificate.'
    $certificatePath = 'Cert:\LocalMachine\My\' + $certificateThumbprint
    if (Test-Path -Path $certificatePath) {
        Remove-Item -Path $certificatePath -Force
    }
}

if ($forceWinRMUninstall) {
    Write-Output 'Force WinRM uninstall mode is enabled; removing listeners and disabling WinRM.'
    Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction SilentlyContinue |
        ForEach-Object { Remove-Item -Path $_.PSPath -Recurse -Force }
    Get-NetFirewallRule -Name 'DevAlchemyLocalWinRMHTTPS' -ErrorAction SilentlyContinue | Remove-NetFirewallRule | Out-Null
    Disable-WinRMFirewallRules
    Restore-WinRMServiceState $false 'Disabled'
} else {
    Write-Output 'Restoring the original WinRM service state.'
    Restore-WinRMServiceState ([bool]$state.WinRMServiceWasRunning) ([string]$state.WinRMServiceStartMode)
}

Write-Output 'Local Windows provision cleanup completed.'
