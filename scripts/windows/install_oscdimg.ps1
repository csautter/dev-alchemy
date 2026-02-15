# Download and install Windows ADK Deployment Tools (includes oscdimg)
# https://learn.microsoft.com/de-de/windows-hardware/get-started/adk-install
$adkUrl = "https://go.microsoft.com/fwlink/?linkid=2289980" # Windows ADK for Windows 11, version 22H2
$adkInstaller = "$env:TEMP\adksetup.exe"

Write-Host "Downloading Windows ADK installer..."
Invoke-WebRequest -Uri $adkUrl -OutFile $adkInstaller

Write-Host "Starting Windows ADK installer (Deployment Tools only)..."
Start-Process -FilePath $adkInstaller -ArgumentList "/features OptionId.DeploymentTools /quiet /norestart" -Wait

# Find oscdimg.exe and add its folder to the PATH for the current session
$oscdimgPath = Get-ChildItem -Path "C:\Program Files (x86)\Windows Kits" -Recurse -Filter oscdimg.exe -ErrorAction SilentlyContinue | Select-Object -First 1

if ($oscdimgPath) {
    $oscdimgDir = $oscdimgPath.DirectoryName
    Write-Host "oscdimg.exe found at $oscdimgDir"
    $env:Path += ";$oscdimgDir"
    Write-Host "Added oscdimg directory to PATH for this session."
} else {
    Write-Host "oscdimg.exe not found. Please check the ADK installation."
}