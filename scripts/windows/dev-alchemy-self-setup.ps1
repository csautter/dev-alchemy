param(
    [switch]$WithGo,
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
# renovate: datasource=nuget depName=python313 versioning=nuget registryUrl=https://community.chocolatey.org/api/v2/
$nativePythonVersion = "3.13.13"
# renovate: datasource=nuget depName=cygwin versioning=nuget registryUrl=https://community.chocolatey.org/api/v2/
$cygwinVersion = "3.6.9"
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
    "libssl-devel",
    "sshpass"
)

function Test-IsAdministrator {
    return ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
        [Security.Principal.WindowsBuiltInRole] "Administrator"
    )
}

function Invoke-SelfElevated {
    $scriptPath = $PSCommandPath
    if (-not $scriptPath) {
        $scriptPath = $MyInvocation.MyCommand.Path
    }
    if (-not $scriptPath) {
        throw "Unable to determine the installer script path for elevation."
    }

    $argumentList = @(
        "-NoLogo",
        "-NoProfile",
        "-ExecutionPolicy", "Bypass",
        "-File", $scriptPath
    )
    if ($WithGo) {
        $argumentList += "-WithGo"
    }
    if ($VirtualBox) {
        $argumentList += "-VirtualBox"
    }

    Write-Output "Requesting administrator privileges through UAC..."

    try {
        $process = Start-Process -FilePath "powershell.exe" -ArgumentList $argumentList -Verb RunAs -WorkingDirectory (Get-Location).Path -Wait -PassThru
    } catch [System.ComponentModel.Win32Exception] {
        if ($_.Exception.NativeErrorCode -eq 1223) {
            throw "Administrator privileges are required to continue. The UAC prompt was cancelled."
        }
        throw
    }

    exit $process.ExitCode
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

    $maxAttempts = 3
    for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
        $chocoOutput = & choco @args 2>&1
        $exitCode = $LASTEXITCODE

        foreach ($line in $chocoOutput) {
            Write-Output $line
        }

        if ($exitCode -in @(0, 1641, 3010)) {
            Refresh-ProcessPath
            return
        }

        $shouldRetry = $attempt -lt $maxAttempts -and (Test-ShouldRetryChocolateyCommand -OutputLines $chocoOutput)
        if (-not $shouldRetry) {
            throw "Chocolatey failed to $command $PackageName $Version with exit code $exitCode."
        }

        $sleepSeconds = 5 * $attempt
        Write-Warning "Chocolatey failed to $command $PackageName $Version with exit code $exitCode. Retrying in $sleepSeconds seconds (attempt $($attempt + 1) of $maxAttempts)..."
        Start-Sleep -Seconds $sleepSeconds
    }
}

function Ensure-GoInstallRootClean {
    param(
        [Parameter(Mandatory = $true)]
        [string]$DesiredVersion
    )

    $goInstallRoot = "C:\Program Files\Go"
    $goExePath = Join-Path $goInstallRoot "bin\go.exe"
    $staleSwissMapPath = Join-Path $goInstallRoot "src\internal\abi\map_swiss.go"
    $installedVersion = Get-ChocolateyInstalledVersion -PackageName "golang"

    $shouldRemove = $false

    if ($installedVersion -and $installedVersion -ne $DesiredVersion) {
        Write-Output "Go $installedVersion detected. Removing the existing GOROOT before installing $DesiredVersion to avoid stale standard-library files."
        $shouldRemove = $true
    } elseif (Test-Path $staleSwissMapPath) {
        Write-Output "A stale Go source file was detected at $staleSwissMapPath. Removing the existing GOROOT before reinstalling Go."
        $shouldRemove = $true
    } elseif ((-not $installedVersion) -and (Test-Path $goExePath)) {
        Write-Output "An unmanaged Go installation was detected at $goInstallRoot. Removing it so the pinned Chocolatey install can start from a clean state."
        $shouldRemove = $true
    }

    if (-not $shouldRemove) {
        return
    }

    if (Test-Path $goInstallRoot) {
        Remove-Item -Path $goInstallRoot -Recurse -Force
    }

    Refresh-ProcessPath
}

function Assert-GoToolchainLayout {
    $goExe = Get-Command go -ErrorAction SilentlyContinue
    if (-not $goExe) {
        throw "Go was not found on PATH after installation."
    }

    $goRoot = (& go env GOROOT).Trim()
    if (-not $goRoot) {
        throw "Unable to determine GOROOT after installing Go."
    }

    $stalePaths = @(
        (Join-Path $goRoot "src\internal\abi\map_swiss.go")
    ) | Where-Object { Test-Path $_ }

    if ($stalePaths.Count -gt 0) {
        $staleList = $stalePaths -join ", "
        throw "Detected stale Go standard-library source files after installation: $staleList. Remove $goRoot and reinstall Go."
    }
}

function Assert-NativePythonAvailable {
    $pythonExe = Get-Command python -ErrorAction SilentlyContinue
    if (-not $pythonExe) {
        throw "Native Windows Python was not found on PATH after installation."
    }
}

function Get-NativePythonChocolateyPackageName {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Version
    )

    $versionMatch = [regex]::Match($Version, "^(?<major>\d+)\.(?<minor>\d+)")
    if (-not $versionMatch.Success) {
        throw "Unable to derive the Chocolatey Python package name from version '$Version'."
    }

    return "python{0}{1}" -f $versionMatch.Groups["major"].Value, $versionMatch.Groups["minor"].Value
}

function Test-ShouldRetryChocolateyCommand {
    param(
        [Parameter(Mandatory = $true)]
        [object[]]$OutputLines
    )

    $combinedOutput = ($OutputLines | Out-String)
    return (
        $combinedOutput -match "Response status code does not indicate success:\s+(408|429|5\d\d)" -or
        $combinedOutput -match "Unable to connect to source" -or
        $combinedOutput -match "The operation has timed out" -or
        $combinedOutput -match "temporarily unavailable" -or
        $combinedOutput -match "The remote name could not be resolved" -or
        $combinedOutput -match "An error occurred while sending the request" -or
        $combinedOutput -match "The request was aborted"
    )
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
    Invoke-SelfElevated
}

Ensure-ChocolateyInstalled

if ($WithGo) {
    Ensure-GoInstallRootClean -DesiredVersion $golangVersion
    Ensure-ChocolateyPackage -PackageName "golang" -Version $golangVersion
    Assert-GoToolchainLayout
} else {
    Write-Output "Skipping Go installation because -WithGo was not specified."
}
Ensure-ChocolateyPackage -PackageName "git" -Version $gitVersion
Ensure-ChocolateyPackage -PackageName "make" -Version $makeVersion
Ensure-ChocolateyPackage -PackageName "packer" -Version $packerVersion
Ensure-ChocolateyPackage -PackageName "azure-cli" -Version $azureCliVersion
$nativePythonPackageName = Get-NativePythonChocolateyPackageName -Version $nativePythonVersion
Ensure-ChocolateyPackage -PackageName $nativePythonPackageName -Version $nativePythonVersion
Assert-NativePythonAvailable
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
python --version
& $bashExePath -lc "python3 --version"
& $bashExePath -lc "ansible --version"
if ($WithGo) {
    go version
}
git --version
make --version
packer version
az version
