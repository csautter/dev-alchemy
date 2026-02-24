# Expects the following environment variables (set by the composite action):
#   CACHE_FILES       - JSON array of cache file descriptors
#   SUBSCRIPTION_ID   - Azure subscription ID
#   LOCAL_CACHE_DIR   - Optional local runner cache directory
[CmdletBinding()]
param()

$files         = $env:CACHE_FILES | ConvertFrom-Json
$storageAcct   = ("ghrunner$env:SUBSCRIPTION_ID" -replace '-', '').Substring(0, 24)
$resourceGroup = "gh-runner-storage-rg"
$localCacheDir = $env:LOCAL_CACHE_DIR

foreach ($f in $files) {
    $localPath = $f.'local-path'
    $blobName  = $f.'blob-name'
    if (-not $blobName) { $blobName = Split-Path -Leaf $localPath }
    $container = $f.'container'
    if (-not $container) { $container = "build-cache" }

    Write-Host "=== Build cache: $localPath [blob=$blobName container=$container] ==="

    # 1. Already present in workspace -> nothing to do
    if (Test-Path $localPath) {
        Write-Host "  [ok] Already present at $localPath - skipping."
        continue
    }

    # 2. Azure Blob Storage -> download
    # (Windows Azure runners usually have no shared-volume cache)
    Write-Host "  [download] Attempting Azure Blob Storage download..."
    $saArgs = @('storage', 'account', 'show', '--name', $storageAcct, '--resource-group', $resourceGroup)
    $saJson = az @saArgs 2>&1
    $sa = $null
    if ($LASTEXITCODE -eq 0) { $sa = $saJson | ConvertFrom-Json }

    if ($sa) {
        $existsArgs = @('storage', 'blob', 'exists', '--account-name', $storageAcct, '--container-name', $container, '--name', $blobName, '--auth-mode', 'login')
        $blobExists = (az @existsArgs | ConvertFrom-Json).exists

        if ($blobExists) {
            $dir = Split-Path -Path $localPath -Parent
            if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
            $dlArgs = @('storage', 'blob', 'download', '--account-name', $storageAcct, '--container-name', $container, '--name', $blobName, '--file', $localPath, '--auth-mode', 'login')
            az @dlArgs
            Write-Host "  [ok] Downloaded $blobName -> $localPath"

            # Save to local runner cache when available.
            if ($localCacheDir -and (Test-Path $localCacheDir -PathType Container)) {
                $cached = Join-Path $localCacheDir $blobName
                if (Test-Path $cached) {
                    Write-Host "  [ok] Already in local runner cache at $cached."
                } else {
                    Write-Host "  [cache] Saving to local runner cache: $cached"
                    Copy-Item -Path $localPath -Destination $cached
                    Write-Host "  [ok] Saved to local runner cache."
                }
            }
        } else {
            Write-Host "  [miss] Blob $blobName not found in container '$container'."
        }
    } else {
        Write-Host "  [miss] Storage account $storageAcct not found."
    }
}
