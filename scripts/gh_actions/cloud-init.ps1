#ps1_sysnative

# write a file that logs if this file was executed
New-Item -Path "C:\" -Name "execution.log" -ItemType "file" -Force
Add-Content -Path "C:\execution.log" -Value "cloud-init-combined.ps1 executed on $(Get-Date)"

# Install Hyper-V
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All -NoRestart
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V-Management-PowerShell -All -NoRestart
# Optionally, restart if required
if ((Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V).State -ne 'Enabled') {
  Write-Host 'Restart required to complete Hyper-V installation.'
  Restart-Computer -Force
  exit
}

# Install Hyper-V
Set-Location C:\actions-runner
.\config.cmd `
  --url $env:REPO_URL `
  --token $env:TOKEN `
  --unattended `
  --ephemeral `
  --runasservice `
  --labels windows,azure,nested