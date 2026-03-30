$ErrorActionPreference = "Stop"

# Download and install Windows ADK Deployment Tools (includes oscdimg)
# https://learn.microsoft.com/windows-hardware/get-started/adk-install
$adkUrl = "https://go.microsoft.com/fwlink/?linkid=2289980" # Windows ADK for Windows 11, version 22H2
$adkInstaller = Join-Path $env:TEMP "adksetup.exe"
$windowsKitsRoot = Join-Path ${env:ProgramFiles(x86)} "Windows Kits"

function Get-OscdimgPath {
    if (-not (Test-Path $windowsKitsRoot)) {
        return $null
    }

    return Get-ChildItem -Path $windowsKitsRoot -Recurse -Filter oscdimg.exe -ErrorAction SilentlyContinue |
        Select-Object -First 1
}

function Add-PathIfMissing {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Directory
    )

    $pathEntries = $env:Path -split ';'
    if ($pathEntries -contains $Directory) {
        Write-Host "oscdimg directory already present in PATH for this session."
        return
    }

    $env:Path += ";$Directory"
    Write-Host "Added oscdimg directory to PATH for this session."
}

function Add-GitHubPathIfAvailable {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Directory
    )

    if (-not $env:GITHUB_PATH) {
        return
    }

    if ((Test-Path $env:GITHUB_PATH) -and (Select-String -Path $env:GITHUB_PATH -SimpleMatch -Pattern $Directory -Quiet)) {
        Write-Host "oscdimg directory already exported to GITHUB_PATH."
        return
    }

    $Directory | Out-File -FilePath $env:GITHUB_PATH -Encoding utf8 -Append
    Write-Host "Exported oscdimg directory to GITHUB_PATH."
}

$oscdimgPath = Get-OscdimgPath
if (-not $oscdimgPath) {
    Write-Host "oscdimg.exe not found. Downloading Windows ADK installer..."
    Invoke-WebRequest -Uri $adkUrl -OutFile $adkInstaller

    Write-Host "Starting Windows ADK installer (Deployment Tools only)..."
    Start-Process -FilePath $adkInstaller -ArgumentList "/features OptionId.DeploymentTools /quiet /norestart" -Wait

    $oscdimgPath = Get-OscdimgPath
}

if (-not $oscdimgPath) {
    Write-Error "oscdimg.exe not found after ADK installation. Please check the ADK installation."
    exit 1
}

$oscdimgDir = $oscdimgPath.DirectoryName
Write-Host "oscdimg.exe found at $oscdimgDir"
Add-PathIfMissing -Directory $oscdimgDir
Add-GitHubPathIfAvailable -Directory $oscdimgDir
