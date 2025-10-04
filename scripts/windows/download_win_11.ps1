param(
    [switch]$GetUrl
)

# Define variables
$FidoVersion = "1.66"
$FidoExe = "Fido.ps1"
$VendorDir = "$PSScriptRoot\..\..\vendor"
$FidoPath = "$VendorDir\$FidoExe"
# LZMA and signature file paths
$FidoLzma = "$VendorDir\Fido.ps1.lzma"
$FidoSig = "$VendorDir\Fido.ps1.lzma.sig"
$SevenZip = "7z.exe" # Assumes 7z.exe is in PATH

$oldLocation = Get-Location

# Create vendor directory if it doesn't exist
if (!(Test-Path $VendorDir)) {
    New-Item -ItemType Directory -Path $VendorDir | Out-Null
}

Set-Location $VendorDir
# Download Fido.ps1.lzma and .sig if not present
if (!(Test-Path $FidoLzma)) {
    Write-Host "Downloading Fido.ps1.lzma..."
    Invoke-WebRequest -Uri "https://github.com/pbatard/Fido/releases/download/v$FidoVersion/Fido.ps1.lzma" -OutFile $FidoLzma
}
if (!(Test-Path $FidoSig)) {
    Write-Host "Downloading Fido.ps1.lzma.sig..."
    Invoke-WebRequest -Uri "https://github.com/pbatard/Fido/releases/download/v$FidoVersion/Fido.ps1.lzma.sig" -OutFile $FidoSig
}

# Verify sha256 checksum of Fido.ps1.lzma sha256:a6d2b028b6b1b022c0e564ecadbab0e1971b42886df9c7de99c074124762ad23
$ExpectedHash = "5674ebbe02e7e9af4ed36bc0ad37d2b5baa23109869bd6b14ebff781ecd27f45"
$FileHash = (Get-FileHash -Path $FidoLzma -Algorithm SHA256).Hash
if ($FileHash -ne $ExpectedHash) {
    Write-Error "Hash mismatch for Fido.ps1.lzma. Expected: $ExpectedHash, Actual: $FileHash"
    exit 1
} else {
    Write-Host "Fido.ps1.lzma hash verified."
}

# Extract Fido.ps1 from Fido.ps1.lzma
if (!(Test-Path $FidoPath) -and (Test-Path $FidoLzma)) {
    Write-Host "Extracting Fido.ps1 from Fido.ps1.lzma..."
    & $SevenZip e $FidoLzma | Out-Null
}

# create vendor/windows directory if it doesn't exist
if (!(Test-Path "$VendorDir\windows")) {
    New-Item -ItemType Directory -Path "$VendorDir\windows" | Out-Null
}

# Run Fido to download Windows 11 ISO with options
Write-Host "Launching Fido to download Windows 11 ISO..."
Set-Location $PSScriptRoot\..\..\vendor\windows\
$FidoArgs = @("-Win", "11", "-Rel", "Latest", "-Ed", "Pro", "-Arch", "x64", "-Lang", "English")
if ($GetUrl) {
    $FidoArgs += @("-GetUrl")
}
powershell -ExecutionPolicy Bypass -File $FidoPath @FidoArgs

if( $GetUrl ) {
    Set-Location $oldLocation
    exit 0
}

# Check if ISO was downloaded
$IsoPath = Get-ChildItem -Path . -Filter "*.iso" | Select-Object -First 1
if ($IsoPath) {
    Write-Host "Windows 11 ISO downloaded successfully: $($IsoPath.FullName)"
} else {
    Write-Error "Failed to download Windows 11 ISO."
    exit 1
}

Set-Location $oldLocation