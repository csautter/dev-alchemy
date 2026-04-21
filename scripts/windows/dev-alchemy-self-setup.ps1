param(
    [switch]$VirtualBox
)

$ErrorActionPreference = "Stop"

# renovate: datasource=nuget depName=golang versioning=nuget registryUrl=https://community.chocolatey.org/api/v2/
$golangVersion = "1.26.2"
# renovate: datasource=nuget depName=git versioning=nuget registryUrl=https://community.chocolatey.org/api/v2/
$gitVersion = "2.54.0"
# renovate: datasource=nuget depName=make versioning=nuget registryUrl=https://community.chocolatey.org/api/v2/
$makeVersion = "4.4.1"
# renovate: datasource=nuget depName=packer versioning=nuget registryUrl=https://community.chocolatey.org/api/v2/
$packerVersion = "1.15.0"
# renovate: datasource=nuget depName=azure-cli versioning=nuget registryUrl=https://community.chocolatey.org/api/v2/
$azureCliVersion = "2.85.0"
# renovate: datasource=nuget depName=cygwin versioning=nuget registryUrl=https://community.chocolatey.org/api/v2/
$cygwinVersion = "3.6.7"
# renovate: datasource=nuget depName=cyg-get versioning=nuget registryUrl=https://community.chocolatey.org/api/v2/
$cygGetVersion = "1.2.2"
# renovate: datasource=nuget depName=virtualbox versioning=nuget registryUrl=https://community.chocolatey.org/api/v2/
$virtualBoxVersion = "7.2.6"
# renovate: datasource=pypi depName=ansible versioning=pep440
$ansibleVersion = "13.6.0"
# renovate: datasource=pypi depName=pywinrm versioning=pep440
$pywinrmVersion = "0.5.0"

$cygwinInstallRoot = "C:\tools\cygwin"
$cygwinPackages = @(
    "python312",
    "python312-pip",
    "python312-cryptography",
    "openssh",
    "git",
    "make",
    "gcc-core",
    "gcc-g++",
    "libffi-devel",
    "openssl-devel",
    "sshpass"
)

function Test-IsAdministrator {
    return ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
        [Security.Principal.WindowsBuiltInRole] "Administrator"
    )
}

function Refresh-ProcessPath {
    $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $env:Path = @($machinePath, $userPath) -join ";"
}

function Ensure-PathContains {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PathEntry
    )

    $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $currentEntries = $machinePath -split ";" | Where-Object { $_ -ne "" }
    if ($currentEntries -contains $PathEntry) {
        return
    }

    [Environment]::SetEnvironmentVariable("Path", ($currentEntries + $PathEntry) -join ";", "Machine")
    Refresh-ProcessPath
}

function Ensure-ChocolateyInstalled {
    if (Get-Command choco -ErrorAction SilentlyContinue) {
        Write-Output "Chocolatey is already installed."
        return
    }

    Write-Output "Chocolatey not found. Installing Chocolatey..."
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    Invoke-Expression ((New-Object System.Net.WebClient).DownloadString("https://community.chocolatey.org/install.ps1"))
    Refresh-ProcessPath
}

function Get-ChocolateyInstalledVersion {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PackageName
    )

    $output = & choco list --local-only --exact $PackageName --limit-output 2>$null
    if ($LASTEXITCODE -ne 0) {
        return $null
    }

    foreach ($line in $output) {
        if ($line -match "^$([regex]::Escape($PackageName))\|(.+)$") {
            return $Matches[1].Trim()
        }
    }

    return $null
}

function Ensure-ChocolateyPackage {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PackageName,
        [Parameter(Mandatory = $true)]
        [string]$Version,
        [string[]]$ExtraArgs = @()
    )

    $installedVersion = Get-ChocolateyInstalledVersion -PackageName $PackageName
    if ($installedVersion -eq $Version) {
        Write-Output "$PackageName $Version is already installed."
        return
    }

    if ($installedVersion) {
        Write-Output "$PackageName $installedVersion detected. Enforcing pinned version $Version..."
        $command = "upgrade"
    } else {
        Write-Output "$PackageName not found. Installing pinned version $Version..."
        $command = "install"
    }

    $args = @($command, $PackageName, "-y", "--no-progress", "--version", $Version)
    if ($command -eq "upgrade") {
        $args += "--allow-downgrade"
    }
    if ($ExtraArgs.Count -gt 0) {
        $args += $ExtraArgs
    }

    & choco @args
    if ($LASTEXITCODE -notin @(0, 1641, 3010)) {
        throw "Chocolatey failed to $command $PackageName $Version with exit code $LASTEXITCODE."
    }

    Refresh-ProcessPath
}

function Get-CygwinRootDir {
    $registryKeys = @(
        "HKLM:\SOFTWARE\Cygwin\setup",
        "HKLM:\SOFTWARE\WOW6432Node\Cygwin\setup"
    )

    foreach ($registryKey in $registryKeys) {
        if (-not (Test-Path $registryKey)) {
            continue
        }

        $rootDir = (Get-ItemProperty -Path $registryKey -Name rootdir -ErrorAction SilentlyContinue).rootdir
        if ($rootDir) {
            return $rootDir
        }
    }

    $fallbackCandidates = @(
        $cygwinInstallRoot,
        "C:\cygwin64",
        "C:\cygwin",
        "D:\cygwin"
    )
    foreach ($candidate in $fallbackCandidates) {
        if (Test-Path (Join-Path $candidate "bin\bash.exe")) {
            return $candidate
        }
    }

    throw "Unable to locate the Cygwin installation root."
}

function Get-CygwinInstalledDbPath {
    return Join-Path (Get-CygwinRootDir) "etc\setup\installed.db"
}

function Get-CygwinBashPath {
    return Join-Path (Get-CygwinRootDir) "bin\bash.exe"
}

function Test-CygwinPackageInstalled {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PackageName
    )

    $installedDb = Get-CygwinInstalledDbPath
    if (-not (Test-Path $installedDb)) {
        throw "installed.db not found at $installedDb"
    }

    return Select-String -Path $installedDb -Pattern "^(?i)$([regex]::Escape($PackageName))\s" -Quiet
}

function Ensure-CygwinPackage {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PackageName
    )

    if (Test-CygwinPackageInstalled -PackageName $PackageName) {
        Write-Output "$PackageName is installed."
        return
    }

    Write-Output "$PackageName is not installed. Installing..."
    & cyg-get $PackageName
    if ($LASTEXITCODE -ne 0) {
        throw "cyg-get failed to install $PackageName with exit code $LASTEXITCODE."
    }
}

function Get-CygwinPythonCommand {
    return "python3"
}

function Get-CygwinPipPackageVersion {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PackageName
    )

    $bashExePath = Get-CygwinBashPath
    $pythonCommand = Get-CygwinPythonCommand
    $output = & $bashExePath -lc "$pythonCommand -m pip show $PackageName 2>/dev/null"
    if ($LASTEXITCODE -ne 0) {
        return $null
    }

    foreach ($line in $output) {
        if ($line -match "^Version:\s+(.+)$") {
            return $Matches[1].Trim()
        }
    }

    return $null
}

function Ensure-CygwinPipPackage {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PackageName,
        [Parameter(Mandatory = $true)]
        [string]$Version
    )

    $installedVersion = Get-CygwinPipPackageVersion -PackageName $PackageName
    if ($installedVersion -eq $Version) {
        Write-Output "$PackageName $Version is already installed in Cygwin."
        return
    }

    $bashExePath = Get-CygwinBashPath
    $pythonCommand = Get-CygwinPythonCommand
    $constraint = "'" + $PackageName + "==" + $Version + "'"

    if ($installedVersion) {
        Write-Output "$PackageName $installedVersion detected in Cygwin. Enforcing pinned version $Version..."
    } else {
        Write-Output "$PackageName not found in Cygwin. Installing pinned version $Version..."
    }

    & $bashExePath -lc "$pythonCommand -m pip install --disable-pip-version-check --upgrade $constraint"
    if ($LASTEXITCODE -ne 0) {
        throw "pip failed to install $PackageName==$Version inside Cygwin with exit code $LASTEXITCODE."
    }
}

if (-not (Test-IsAdministrator)) {
    throw "This script must be run as an Administrator."
}

Ensure-ChocolateyInstalled

Ensure-ChocolateyPackage -PackageName "golang" -Version $golangVersion
Ensure-ChocolateyPackage -PackageName "git" -Version $gitVersion
Ensure-ChocolateyPackage -PackageName "make" -Version $makeVersion
Ensure-ChocolateyPackage -PackageName "packer" -Version $packerVersion
Ensure-ChocolateyPackage -PackageName "azure-cli" -Version $azureCliVersion
Ensure-ChocolateyPackage -PackageName "cygwin" -Version $cygwinVersion -ExtraArgs @("--params", "`"/InstallDir:$cygwinInstallRoot /NoStartMenu`"")
Ensure-ChocolateyPackage -PackageName "cyg-get" -Version $cygGetVersion

if ($VirtualBox) {
    Ensure-ChocolateyPackage -PackageName "virtualbox" -Version $virtualBoxVersion
    Ensure-PathContains -PathEntry "C:\Program Files\Oracle\VirtualBox"
} else {
    Write-Output "Skipping VirtualBox installation because -VirtualBox was not specified."
}

Ensure-PathContains -PathEntry "C:\Program Files\Git\bin"

foreach ($package in $cygwinPackages) {
    Ensure-CygwinPackage -PackageName $package
}

Ensure-CygwinPipPackage -PackageName "ansible" -Version $ansibleVersion
Ensure-CygwinPipPackage -PackageName "pywinrm" -Version $pywinrmVersion

$bashExePath = Get-CygwinBashPath
& $bashExePath -lc "python3 --version"
& $bashExePath -lc "ansible --version"
go version
git --version
make --version
packer version
az version
