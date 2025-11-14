# Check for administrative privileges
if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script must be run as an Administrator."
    exit 1
}

# check if chocolatey is installed
if (-not (Get-Command choco -ErrorAction SilentlyContinue)) {
    Write-Output "Chocolatey not found. Installing Chocolatey..."
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
} else {
    Write-Output "Chocolatey is already installed."
}

# Install Cygwin
if (-not (Get-Command cygwin -ErrorAction SilentlyContinue)) {
    Write-Output "Cygwin not found. Installing Cygwin..."
    choco install cygwin -y
} else {
    Write-Output "Cygwin is already installed."
}

# Install cyg-get
if (-not (Get-Command cyg-get -ErrorAction SilentlyContinue)) {
    Write-Output "cyg-get not found. Installing cyg-get..."
    choco install -y cyg-get
} else {
    Write-Output "cyg-get is already installed."
}

# Install required Cygwin packages
$installedDb = "C:\tools\cygwin\etc\setup\installed.db"

function Test-CygwinPackageInstalled {
    param(
        [string]$PackageName
    )

    if (-Not (Test-Path $installedDb)) {
        Write-Error "installed.db not found at $installedDb"
        return $false
    }

    $found = Select-String -Path $installedDb -Pattern "^(?i)$PackageName\s" -Quiet
    return $found
}

$packages = @(
    "python39",
    "python39-pip",
    "python39-cryptography",
    "openssh",
    "git",
    "make",
    "gcc-core",
    "gcc-g\+\+",
    "libffi-devel",
    "libssl-devel",
    "sshpass"
)
foreach ($package in $packages) {
    if (Test-CygwinPackageInstalled -PackageName $package) {
        Write-Output "$package is installed"
    } else {
        Write-Output "$package is not installed"
        cyg-get $package
    }
}

# Install ansible and pywinrm within Cygwin with pip
$cygwinPipPackages = @("ansible", "pywinrm")
$bashExePath = "C:\tools\cygwin\bin\bash.exe"

foreach ($pkg in $cygwinPipPackages) {
    if (-not (& $bashExePath -c "pip3 show $pkg" | Select-String -Pattern "Name: $pkg")) {
        Write-Output "$pkg not found. Installing $pkg..."
        & $bashExePath -c "pip3 install $pkg"
    } else {
        Write-Output "$pkg is already installed."
    }
}

# Verify installations
& $bashExePath -c "ansible --version"