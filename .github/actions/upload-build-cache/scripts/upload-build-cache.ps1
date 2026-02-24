# Expects the following environment variables (set by the composite action):
#   CACHE_FILES       - JSON array of cache file descriptors
#   SUBSCRIPTION_ID   - Azure subscription ID
[CmdletBinding()]
param()

$files       = $env:CACHE_FILES | ConvertFrom-Json
$storageAcct = ("ghrunner$env:SUBSCRIPTION_ID" -replace '-', '').Substring(0, 24)

foreach ($f in $files) {
    $localPath = $f.'local-path'
    $blobName  = $f.'blob-name'
    if (-not $blobName) { $blobName = Split-Path -Leaf $localPath }
    $container = $f.'container'
    if (-not $container) { $container = "build-cache" }

    Write-Host "=== Build cache upload: $localPath [blob=$blobName container=$container] ==="

    if (-not (Test-Path $localPath)) {
        Write-Host "  ✗ File not found at $localPath — skipping."
        continue
    }

    $existsArgs = @('storage', 'blob', 'exists', '--account-name', $storageAcct, '--container-name', $container, '--name', $blobName, '--auth-mode', 'login', '--query', 'exists', '--output', 'tsv')
    $blobExists = az @existsArgs

    if ($blobExists -eq "true") {
        Write-Host "  ✓ Blob $blobName already exists in '$container' — skipping upload."
    } else {
        Write-Host "  ↑ Uploading $blobName to container '$container'..."
        $uploadArgs = @('storage', 'blob', 'upload', '--account-name', $storageAcct, '--container-name', $container, '--name', $blobName, '--file', $localPath, '--auth-mode', 'login', '--overwrite', 'false')
        az @uploadArgs
        Write-Host "  ✓ Uploaded $blobName."
    }
}
