# Expects the following environment variables (set by the composite action):
#   CACHE_FILES       - JSON array of cache file descriptors
#   SUBSCRIPTION_ID   - Azure subscription ID
#   LOCAL_CACHE_DIR   - Optional local runner cache directory
[CmdletBinding()]
param()

$files         = $env:CACHE_FILES | ConvertFrom-Json
$storageAcct   = ("ghrunner$env:SUBSCRIPTION_ID" -replace '-', '').Substring(0, 24)
$localCacheDir = $env:LOCAL_CACHE_DIR

foreach ($f in $files) {
    $localPath = $f.'local-path'
    $blobName  = $f.'blob-name'
    if (-not $blobName) { $blobName = Split-Path -Leaf $localPath }
    $container = $f.'container'
    if (-not $container) { $container = "build-cache" }

    Write-Host "=== Build cache upload: $localPath [blob=$blobName container=$container] ==="

    if (-not (Test-Path $localPath -PathType Leaf)) {
        Write-Host "  [miss] File not found at $localPath - skipping."
        continue
    }

    # 1. Save to local runner cache when available.
    if ($localCacheDir -and (Test-Path $localCacheDir -PathType Container)) {
        $cached = Join-Path $localCacheDir $blobName
        if (Test-Path $cached -PathType Leaf) {
            Write-Host "  [ok] Already in local runner cache at $cached."
        } else {
            Write-Host "  [cache] Saving to local runner cache: $cached"
            Copy-Item -Path $localPath -Destination $cached
            Write-Host "  [ok] Saved to local runner cache."
        }
    }

    # 2. Upload to Azure Blob Storage (skip if blob already exists)
    $existsArgs = @('storage', 'blob', 'exists', '--account-name', $storageAcct, '--container-name', $container, '--name', $blobName, '--auth-mode', 'login', '--query', 'exists', '--output', 'tsv')
    $blobExists = az @existsArgs

    if ($blobExists -eq "true") {
        Write-Host "  [ok] Blob $blobName already exists in '$container' - skipping upload."
    } else {
        Write-Host "  [upload] Uploading $blobName to container '$container'..."
        $uploadArgs = @('storage', 'blob', 'upload', '--account-name', $storageAcct, '--container-name', $container, '--name', $blobName, '--file', $localPath, '--auth-mode', 'login', '--overwrite', 'false')
        az @uploadArgs
        Write-Host "  [ok] Uploaded $blobName."
    }
}
