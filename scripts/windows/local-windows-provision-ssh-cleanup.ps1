$ErrorActionPreference = 'Stop'

$statePath = $env:DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH
$forceSSHUninstall = [System.Convert]::ToBoolean($env:DEV_ALCHEMY_LOCAL_WINDOWS_FORCE_SSH_UNINSTALL)
$openSSHBuiltInFirewallRuleName = 'OpenSSH-Server-In-TCP'
$localFirewallRuleName = 'DevAlchemyLocalSSHDLoopback'
$defaultShellRegistryPath = 'HKLM:\SOFTWARE\OpenSSH'
$defaultShellRegistryName = 'DefaultShell'
$administratorsAuthorizedKeysPath = Join-Path $env:ProgramData 'ssh\administrators_authorized_keys'
$administratorsSid = '*S-1-5-32-544'
$systemSid = '*S-1-5-18'

if ([string]::IsNullOrWhiteSpace($statePath) -or -not (Test-Path -Path $statePath)) {
    Write-Output 'No local Windows SSH provision state file was found; skipping cleanup.'
    return
}

Write-Output 'Loading local Windows SSH provision cleanup state.'
$state = Get-Content -Path $statePath -Raw | ConvertFrom-Json

function Restore-ServiceState([string]$name, [bool]$wasRunning, [string]$startMode) {
    $service = Get-Service -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $service) {
        return
    }

    if ($wasRunning) {
        if ($service.Status -ne 'Running') {
            Start-Service -Name $name
            $service = Get-Service -Name $name
        }
    } elseif ($service.Status -eq 'Running') {
        Stop-Service -Name $name -Force
        $service = Get-Service -Name $name
    }

    switch ($startMode.ToLowerInvariant()) {
        'auto' {
            Set-Service -Name $name -StartupType Automatic
        }
        'manual' {
            Set-Service -Name $name -StartupType Manual
        }
        'disabled' {
            if ($service.Status -eq 'Running') {
                Stop-Service -Name $name -Force
            }
            Set-Service -Name $name -StartupType Disabled
        }
    }
}

function Disable-ServiceState([string]$name) {
    $service = Get-Service -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $service) {
        return
    }

    if ($service.Status -eq 'Running') {
        Stop-Service -Name $name -Force
    }

    Set-Service -Name $name -StartupType Disabled
}

function Restore-NetFirewallRuleState([string]$name, [bool]$existed, [bool]$enabled) {
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

function Restore-LoopbackFirewallRuleState([string]$name, [bool]$existed, [bool]$enabled, [string]$localPort) {
    Get-NetFirewallRule -Name $name -ErrorAction SilentlyContinue | Remove-NetFirewallRule | Out-Null

    if (-not $existed) {
        return
    }
    if ([string]::IsNullOrWhiteSpace($localPort)) {
        return
    }

    New-NetFirewallRule -Name $name -DisplayName 'Dev Alchemy Local OpenSSH Loopback' -Direction Inbound -Action Allow -Protocol TCP -LocalAddress 127.0.0.1 -LocalPort $localPort -Profile Any | Out-Null
    if (-not $enabled) {
        Disable-NetFirewallRule -Name $name | Out-Null
    }
}

function Restore-RegistryStringState([string]$path, [string]$name, [bool]$existed, [string]$value) {
    if ($existed) {
        if (-not (Test-Path -Path $path)) {
            New-Item -Path $path -Force | Out-Null
        }
        New-ItemProperty -Path $path -Name $name -Value $value -PropertyType String -Force | Out-Null
        return
    }

    if (Test-Path -Path $path) {
        Remove-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
    }
}

function Restore-FileState([string]$path, [bool]$existed, [string]$contentBase64, [string]$sddl) {
    if ([string]::IsNullOrWhiteSpace($path)) {
        return
    }

    if (Test-Path -Path $path) {
        Grant-AdministrativePathAccess $path
        $parentPath = Split-Path -Parent $path
        if (-not [string]::IsNullOrWhiteSpace($parentPath) -and (Test-Path -Path $parentPath)) {
            Grant-AdministrativePathAccess $parentPath
        }
    }

    if ($existed) {
        $directory = Split-Path -Parent $path
        if (-not (Test-Path -Path $directory)) {
            New-Item -Path $directory -ItemType Directory -Force | Out-Null
        }

        $bytes = @()
        if (-not [string]::IsNullOrWhiteSpace($contentBase64)) {
            $bytes = [System.Convert]::FromBase64String($contentBase64)
        }
        [System.IO.File]::WriteAllBytes($path, $bytes)

        if (-not [string]::IsNullOrWhiteSpace($sddl)) {
            $acl = New-Object System.Security.AccessControl.FileSecurity
            $acl.SetSecurityDescriptorSddlForm($sddl)
            Set-Acl -Path $path -AclObject $acl
        }
        return
    }

    if (Test-Path -Path $path) {
        Remove-Item -Path $path -Force
    }
}

function Grant-AdministrativePathAccess([string]$path) {
    if ([string]::IsNullOrWhiteSpace($path) -or -not (Test-Path -Path $path)) {
        return
    }

    $takeownArgs = @('/A', '/F', $path)
    $icaclsArgs = @($path, '/grant', ($administratorsSid + ':F'), ($systemSid + ':F'))
    $null = & takeown.exe @takeownArgs 2>$null
    $null = & icacls.exe @icaclsArgs 2>$null
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

$userName = [string]$state.UserName
$administratorsGroupName = Get-LocalAdministratorsGroupName
if (-not [string]::IsNullOrWhiteSpace($userName)) {
    if ($forceSSHUninstall) {
        Write-Output 'Force SSH uninstall mode is enabled; removing the local Ansible account.'
        $localUser = Get-LocalUser -Name $userName -ErrorAction SilentlyContinue
        if ($null -ne $localUser) {
            if (Test-IsLocalAdministrator $administratorsGroupName $userName) {
                Remove-LocalGroupMember -Group $administratorsGroupName -Member $userName -ErrorAction SilentlyContinue
            }
            Remove-LocalUser -Name $userName
        }
    } elseif ([bool]$state.UserExisted) {
        Write-Output 'Restoring the original local Ansible account state.'
        if (-not [bool]$state.UserWasAdministrator -and (Test-IsLocalAdministrator $administratorsGroupName $userName)) {
            Remove-LocalGroupMember -Group $administratorsGroupName -Member $userName -ErrorAction SilentlyContinue
        }

        $localUser = Get-LocalUser -Name $userName -ErrorAction SilentlyContinue
        if ($null -ne $localUser) {
            Set-LocalUser -Name $userName -Description ([string]$state.UserDescription)
            if ([bool]$state.UserWasEnabled) {
                Enable-LocalUser -Name $userName
            } else {
                Disable-LocalUser -Name $userName
            }
        }
    } else {
        Write-Output 'Removing the temporary local Ansible account.'
        $localUser = Get-LocalUser -Name $userName -ErrorAction SilentlyContinue
        if ($null -ne $localUser) {
            if (Test-IsLocalAdministrator $administratorsGroupName $userName) {
                Remove-LocalGroupMember -Group $administratorsGroupName -Member $userName -ErrorAction SilentlyContinue
            }
            Remove-LocalUser -Name $userName
        }
    }
}

Write-Output 'Restoring the administrator and per-user authorized_keys files plus OpenSSH shell configuration.'
Restore-FileState $administratorsAuthorizedKeysPath ([bool]$state.AuthorizedKeysExisted) ([string]$state.AuthorizedKeysContentBase64) ([string]$state.AuthorizedKeysSddl)
Restore-FileState ([string]$state.UserAuthorizedKeysPath) ([bool]$state.UserAuthorizedKeysExisted) ([string]$state.UserAuthorizedKeysContentBase64) ([string]$state.UserAuthorizedKeysSddl)
Restore-RegistryStringState $defaultShellRegistryPath $defaultShellRegistryName ([bool]$state.DefaultShellExisted) ([string]$state.DefaultShellValue)
Restore-FileState (Join-Path $env:ProgramData 'ssh\sshd_config') ([bool]$state.SshdConfigExisted) ([string]$state.SshdConfigContentBase64) ([string]$state.SshdConfigSddl)

Write-Output 'Restoring OpenSSH firewall rules.'
Restore-LoopbackFirewallRuleState $localFirewallRuleName ([bool]$state.LocalFirewallRuleExisted) ([bool]$state.LocalFirewallRuleEnabled) ([string]$state.LocalFirewallRulePort)
Restore-NetFirewallRuleState $openSSHBuiltInFirewallRuleName ([bool]$state.BuiltInFirewallRuleExisted) ([bool]$state.BuiltInFirewallRuleEnabled)

if ($forceSSHUninstall) {
    Write-Output 'Force SSH uninstall mode is enabled; disabling sshd and removing SSH firewall rules without uninstalling OpenSSH Server to avoid requiring a reboot.'
    Get-NetFirewallRule -Name $localFirewallRuleName -ErrorAction SilentlyContinue | Remove-NetFirewallRule | Out-Null
    Get-NetFirewallRule -Name $openSSHBuiltInFirewallRuleName -ErrorAction SilentlyContinue | Remove-NetFirewallRule | Out-Null
    Disable-ServiceState 'sshd'
} elseif ([bool]$state.CapabilityInstalled) {
    Write-Output 'Restoring the original sshd service state.'
    Restore-ServiceState 'sshd' ([bool]$state.SshdServiceWasRunning) ([string]$state.SshdServiceStartMode)
} elseif ([bool]$state.CapabilityInstallManaged) {
    Write-Output 'Disabling the OpenSSH Server service that was installed for provisioning so cleanup does not require a reboot.'
    Disable-ServiceState 'sshd'
} else {
    Write-Output ('Leaving the OpenSSH Server capability state unchanged because it started in state "' + [string]$state.CapabilityState + '" and was not installed by this provisioning run.')
    if ([bool]$state.CapabilityPending) {
        Write-Output ('Windows still reports a pending OpenSSH capability change (' + [string]$state.CapabilityState + '). Reboot Windows to finish that pending change before relying on future SSH setup behavior.')
    }
}

Write-Output 'Local Windows SSH provision cleanup completed.'
