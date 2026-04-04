$ErrorActionPreference = 'Stop'

$statePath = $env:DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH
$userName = $env:DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_USER
$passwordPlain = $env:DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_PASSWORD
$forceWinRMUninstall = [System.Convert]::ToBoolean($env:DEV_ALCHEMY_LOCAL_WINDOWS_FORCE_WINRM_UNINSTALL)

if ([string]::IsNullOrWhiteSpace($statePath)) {
    throw 'DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH is required.'
}
if ([string]::IsNullOrWhiteSpace($userName)) {
    throw 'DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_USER is required.'
}
if ([string]::IsNullOrWhiteSpace($passwordPlain)) {
    throw 'DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_PASSWORD is required.'
}

Write-Output 'Validating local Windows provision bootstrap inputs.'

function Save-State($state) {
    $state | ConvertTo-Json -Compress | Set-Content -Path $statePath -Encoding Ascii
}

function Get-WsmanBoolState([string]$path) {
    if (-not (Test-Path -Path $path)) {
        return @{
            Exists = $false
            Value = $false
        }
    }

    return @{
        Exists = $true
        Value = [System.Convert]::ToBoolean((Get-Item -Path $path).Value)
    }
}

function Assert-WsmanPathExists([string]$path) {
    if (-not (Test-Path -Path $path)) {
        throw ("Required WSMan path was not found after preparing WinRM: " + $path)
    }
}

function Get-WinRMServiceState() {
    $service = Get-CimInstance -ClassName Win32_Service -Filter "Name='WinRM'" -ErrorAction Stop
    if ($null -eq $service) {
        throw 'WinRM service is not installed on this host.'
    }

    return @{
        WasRunning = ([string]$service.State -eq 'Running')
        StartMode = [string]$service.StartMode
    }
}

function Get-RegistryDWORDState([string]$path, [string]$name) {
    if (-not (Test-Path -Path $path)) {
        return @{
            Exists = $false
            Value = 0
        }
    }

    $property = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $property) {
        return @{
            Exists = $false
            Value = 0
        }
    }

    return @{
        Exists = $true
        Value = [int]($property.$name)
    }
}

function Set-RegistryDWORDValue([string]$path, [string]$name, [int]$value) {
    if (-not (Test-Path -Path $path)) {
        New-Item -Path $path -Force | Out-Null
    }

    if ($null -eq (Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue)) {
        New-ItemProperty -Path $path -Name $name -Value $value -PropertyType DWord -Force | Out-Null
    } else {
        Set-ItemProperty -Path $path -Name $name -Value $value
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

$listenerKeys = @()
try {
    $listenerKeys = @(Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction Stop | ForEach-Object { $_.Keys })
} catch {
    $listenerKeys = @()
}

$winRMServiceState = Get-WinRMServiceState
$basicAuthState = Get-WsmanBoolState 'WSMan:\localhost\Service\Auth\Basic'
$allowUnencryptedState = Get-WsmanBoolState 'WSMan:\localhost\Service\AllowUnencrypted'
$localAccountTokenFilterPolicyState = Get-RegistryDWORDState 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' 'LocalAccountTokenFilterPolicy'
$firewallRuleState = Get-NetFirewallRuleState 'DevAlchemyLocalWinRMHTTPS'
$administratorsGroupName = Get-LocalAdministratorsGroupName
$userState = Get-ManagedLocalUserState $userName $administratorsGroupName

$state = @{
    UserName = $userName
    UserExisted = [bool]$userState.Exists
    UserWasEnabled = [bool]$userState.Enabled
    UserDescription = [string]$userState.Description
    UserWasAdministrator = [bool]$userState.WasAdministrator
    ForceWinRMUninstall = [bool]$forceWinRMUninstall
    ListenerKeys = @($listenerKeys)
    HadListeners = ($listenerKeys.Count -gt 0)
    WinRMServiceWasRunning = [bool]$winRMServiceState.WasRunning
    WinRMServiceStartMode = [string]$winRMServiceState.StartMode
    BasicAuthPathExisted = [bool]$basicAuthState.Exists
    BasicAuthEnabled = [bool]$basicAuthState.Value
    AllowUnencryptedPathExisted = [bool]$allowUnencryptedState.Exists
    AllowUnencryptedEnabled = [bool]$allowUnencryptedState.Value
    LocalAccountTokenFilterPolicyExisted = [bool]$localAccountTokenFilterPolicyState.Exists
    LocalAccountTokenFilterPolicyValue = [int]$localAccountTokenFilterPolicyState.Value
    FirewallRuleExisted = [bool]$firewallRuleState.Exists
    FirewallRuleEnabled = [bool]$firewallRuleState.Enabled
    CreatedHttpsListener = $false
    CreatedCertificateThumbprint = ''
}
Save-State $state
Write-Output 'Captured existing WinRM, firewall, and local policy state.'

Write-Output 'Ensuring the WinRM service is enabled for local provisioning.'
if ([string]$state.WinRMServiceStartMode -eq 'Disabled') {
    Set-Service -Name 'WinRM' -StartupType Manual
}
if (-not $state.WinRMServiceWasRunning) {
    Start-Service -Name 'WinRM'
}

Write-Output 'Configuring WinRM authentication and local token filter policy.'
Assert-WsmanPathExists 'WSMan:\localhost\Service\AllowUnencrypted'
Assert-WsmanPathExists 'WSMan:\localhost\Service\Auth\Basic'
Set-Item -Path 'WSMan:\localhost\Service\AllowUnencrypted' -Value $false
Set-Item -Path 'WSMan:\localhost\Service\Auth\Basic' -Value $true
Set-RegistryDWORDValue 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' 'LocalAccountTokenFilterPolicy' 1

$httpsListener = @(
    Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction SilentlyContinue |
        Where-Object { $_.Keys -match 'Transport=HTTPS' -and $_.Keys -match 'Port=5986' }
)
if ($httpsListener.Count -eq 0) {
    Write-Output 'Creating a localhost WinRM HTTPS listener and certificate.'
    $certificate = New-SelfSignedCertificate -DnsName 'localhost' -CertStoreLocation 'Cert:\LocalMachine\My' -FriendlyName 'Dev Alchemy Local WinRM HTTPS'
    New-Item -Path WSMan:\localhost\Listener -Transport HTTPS -Address IP:127.0.0.1 -CertificateThumbprint $certificate.Thumbprint -Port 5986 -Force | Out-Null
    $state.CreatedHttpsListener = $true
    $state.CreatedCertificateThumbprint = $certificate.Thumbprint
    Save-State $state
} else {
    Write-Output 'Reusing the existing localhost WinRM HTTPS listener.'
}

if (-not [bool]$state.FirewallRuleExisted) {
    Write-Output 'Creating the local WinRM HTTPS firewall rule.'
    New-NetFirewallRule -Name 'DevAlchemyLocalWinRMHTTPS' -DisplayName 'Dev Alchemy Local WinRM HTTPS' -Direction Inbound -Action Allow -Protocol TCP -LocalAddress 127.0.0.1 -LocalPort 5986 -Profile Any | Out-Null
} elseif (-not [bool]$state.FirewallRuleEnabled) {
    Write-Output 'Enabling the existing local WinRM HTTPS firewall rule.'
    Enable-NetFirewallRule -Name 'DevAlchemyLocalWinRMHTTPS' | Out-Null
} else {
    Write-Output 'Keeping the existing local WinRM HTTPS firewall rule enabled.'
}

$securePassword = ConvertTo-SecureString -String $passwordPlain -AsPlainText -Force
$null = Ensure-ManagedLocalUserForProvisioning $userName $securePassword 'Dev Alchemy Ansible acct' $administratorsGroupName

Write-Output 'Local Windows provision bootstrap completed.'
