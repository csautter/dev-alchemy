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

function Get-ManagedLocalUserState([string]$name, [string]$groupName) {
    $localUser = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $localUser) {
        return @{
            Exists = $false
            Enabled = $false
            Description = ''
            Sid = ''
            WasAdministrator = $false
        }
    }

    return @{
        Exists = $true
        Enabled = [bool]$localUser.Enabled
        Description = [string]$localUser.Description
        Sid = [string]$localUser.SID.Value
        WasAdministrator = Test-IsLocalAdministrator $groupName $name
    }
}

function Ensure-ManagedLocalUserForProvisioning([string]$name, [securestring]$password, [string]$description, [string]$groupName) {
    Write-Output 'Creating or updating the temporary local Ansible account.'

    $localUser = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $localUser) {
        Write-Output ('Creating the temporary local user "' + $name + '".')
        $localUser = New-LocalUser -Name $name -Password $password -PasswordNeverExpires -Description $description
    } else {
        Write-Output ('Reusing the existing local user "' + $name + '" (enabled=' + [string][bool]$localUser.Enabled + ', description=' + [string]$localUser.Description + ').')
        Set-LocalUser -Name $name -Password $password -Description $description
        Enable-LocalUser -Name $name
        $localUser = Get-LocalUser -Name $name
    }

    Write-Output 'Ensuring the temporary local Ansible account is an administrator.'
    if (-not (Test-IsLocalAdministrator $groupName $name)) {
        Add-LocalGroupMember -Group $groupName -Member $name -ErrorAction SilentlyContinue
        Write-Output ('Added "' + $name + '" to the local Administrators group "' + $groupName + '".')
    } else {
        Write-Output ('The local user "' + $name + '" is already a member of "' + $groupName + '".')
    }

    return $localUser
}

function Remove-ManagedLocalUserIfPresent([string]$groupName, [string]$name) {
    $localUser = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $localUser) {
        return
    }

    if (Test-IsLocalAdministrator $groupName $name) {
        Remove-LocalGroupMember -Group $groupName -Member $name -ErrorAction SilentlyContinue
    }
    Remove-LocalUser -Name $name
}

function Restore-ManagedLocalUserState([string]$groupName, $savedState, [string]$name) {
    $localUser = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $localUser) {
        return
    }

    if ([bool]$savedState.UserWasAdministrator) {
        if (-not (Test-IsLocalAdministrator $groupName $name)) {
            Add-LocalGroupMember -Group $groupName -Member $name -ErrorAction SilentlyContinue
        }
    } elseif (Test-IsLocalAdministrator $groupName $name) {
        Remove-LocalGroupMember -Group $groupName -Member $name -ErrorAction SilentlyContinue
    }

    Set-LocalUser -Name $name -Description ([string]$savedState.UserDescription)
    if ([bool]$savedState.UserWasEnabled) {
        Enable-LocalUser -Name $name
    } else {
        Disable-LocalUser -Name $name
    }
}

function Get-ManagedLocalUserCleanupPlan([bool]$userExisted, [bool]$forceManagedAccessUninstall, [string]$transportName) {
    # Force uninstall only tears down the access created for provisioning.
    # A pre-existing local user is always restored; only the temporary user created
    # for provisioning is removed during cleanup.
    if ($userExisted) {
        if ($forceManagedAccessUninstall) {
            return @{
                RestoreUser = $true
                Message = ('Force ' + $transportName + ' uninstall mode is enabled; preserving the pre-existing local Ansible account and restoring its original state.')
            }
        }

        return @{
            RestoreUser = $true
            Message = 'Restoring the original local Ansible account state.'
        }
    }

    if ($forceManagedAccessUninstall) {
        return @{
            RestoreUser = $false
            Message = ('Force ' + $transportName + ' uninstall mode is enabled; removing the temporary local Ansible account created for provisioning.')
        }
    }

    return @{
        RestoreUser = $false
        Message = 'Removing the temporary local Ansible account.'
    }
}
