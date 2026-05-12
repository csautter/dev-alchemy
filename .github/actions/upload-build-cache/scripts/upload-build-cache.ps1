# Expects the following environment variables (set by the composite action):
#   CACHE_FILES       - JSON array of cache file descriptors
#   CACHE_BACKEND     - Remote cache backend: hetzner-s3 or azure
#   SUBSCRIPTION_ID   - Azure subscription ID (azure backend only)
#   HETZNER_S3_*      - Hetzner S3 endpoint, bucket, access key, secret key, and optional prefix
#   LOCAL_CACHE_DIR   - Optional local runner cache directory
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$files          = $env:CACHE_FILES | ConvertFrom-Json
$cacheBackend   = $env:CACHE_BACKEND
if ([string]::IsNullOrWhiteSpace($cacheBackend)) { $cacheBackend = 'hetzner-s3' }
$cacheBackend   = $cacheBackend.ToLowerInvariant()
$localCacheDir  = $env:LOCAL_CACHE_DIR
$script:McAlias = 'dev-alchemy-build-cache'
$script:McBin   = $null
$script:McReady = $false

function Write-Fail {
    param([Parameter(Mandatory = $true)][string]$Message)
    Write-Error "  [error] $Message"
    throw $Message
}

function Normalize-EndpointUrl {
    param([Parameter(Mandatory = $true)][string]$Endpoint)

    if ($Endpoint.StartsWith('http://') -or $Endpoint.StartsWith('https://')) {
        return $Endpoint
    }

    return "https://$Endpoint"
}

function Get-AzureStorageAccount {
    if ([string]::IsNullOrWhiteSpace($env:SUBSCRIPTION_ID)) {
        Write-Fail "subscription-id is required when storage-backend is 'azure'."
    }

    $name = ("ghrunner$($env:SUBSCRIPTION_ID)" -replace '[^A-Za-z0-9]', '').ToLowerInvariant()
    if ($name.Length -gt 24) {
        $name = $name.Substring(0, 24)
    }

    return $name
}

function Ensure-MinioClient {
    if ($script:McBin) {
        return
    }

    $existing = Get-Command mc -ErrorAction SilentlyContinue
    if ($existing) {
        $script:McBin = $existing.Source
        return
    }

    $installRoot = $env:RUNNER_TEMP
    if ([string]::IsNullOrWhiteSpace($installRoot)) { $installRoot = $env:TEMP }
    if ([string]::IsNullOrWhiteSpace($installRoot)) { $installRoot = [System.IO.Path]::GetTempPath() }

    $installDir = Join-Path $installRoot 'dev-alchemy-build-cache'
    if (-not (Test-Path $installDir -PathType Container)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    $script:McBin = Join-Path $installDir 'mc.exe'
    if (-not (Test-Path $script:McBin -PathType Leaf)) {
        $arch = 'amd64'
        if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'ARM64') {
            $arch = 'arm64'
        }

        Write-Host '  [setup] Installing MinIO client for Hetzner S3 cache operations...'
        Invoke-WebRequest -Uri "https://dl.min.io/client/mc/release/windows-$arch/mc.exe" -OutFile $script:McBin -UseBasicParsing
    }
}

function Initialize-HetznerS3 {
    if ($script:McReady) {
        return
    }

    $missing = @()
    if ([string]::IsNullOrWhiteSpace($env:HETZNER_S3_ENDPOINT_URL)) { $missing += 'HETZNER_S3_ENDPOINT_URL' }
    if ([string]::IsNullOrWhiteSpace($env:HETZNER_S3_ACCESS_KEY_ID)) { $missing += 'HETZNER_S3_ACCESS_KEY_ID' }
    if ([string]::IsNullOrWhiteSpace($env:HETZNER_S3_SECRET_ACCESS_KEY)) { $missing += 'HETZNER_S3_SECRET_ACCESS_KEY' }
    if ($missing.Count -gt 0) {
        Write-Fail "Missing Hetzner S3 configuration: $($missing -join ', ')."
    }

    Ensure-MinioClient

    $configRoot = $env:MC_CONFIG_DIR
    if ([string]::IsNullOrWhiteSpace($configRoot)) {
        $base = $env:RUNNER_TEMP
        if ([string]::IsNullOrWhiteSpace($base)) { $base = $env:TEMP }
        if ([string]::IsNullOrWhiteSpace($base)) { $base = [System.IO.Path]::GetTempPath() }
        $configRoot = Join-Path $base 'dev-alchemy-build-cache-mc'
        $env:MC_CONFIG_DIR = $configRoot
    }
    if (-not (Test-Path $configRoot -PathType Container)) {
        New-Item -ItemType Directory -Path $configRoot -Force | Out-Null
    }

    $endpoint = Normalize-EndpointUrl $env:HETZNER_S3_ENDPOINT_URL
    & $script:McBin alias set $script:McAlias $endpoint $env:HETZNER_S3_ACCESS_KEY_ID $env:HETZNER_S3_SECRET_ACCESS_KEY --api S3v4 --path auto *> $null
    if ($LASTEXITCODE -ne 0) {
        Write-Fail 'Failed to configure MinIO client alias for Hetzner S3.'
    }

    $script:McReady = $true
}

function Get-S3BucketForContainer {
    param([Parameter(Mandatory = $true)][string]$Container)

    if (-not [string]::IsNullOrWhiteSpace($env:HETZNER_S3_BUCKET)) {
        return $env:HETZNER_S3_BUCKET
    }

    return $Container
}

function Get-S3KeyForBlob {
    param([Parameter(Mandatory = $true)][string]$BlobName)

    $key = $BlobName.TrimStart('/')
    $prefix = $env:HETZNER_S3_PREFIX
    if (-not [string]::IsNullOrWhiteSpace($prefix)) {
        $prefix = $prefix.Trim('/').Trim()
        if (-not [string]::IsNullOrWhiteSpace($prefix)) {
            return "$prefix/$key"
        }
    }

    return $key
}

function Get-S3RemotePath {
    param(
        [Parameter(Mandatory = $true)][string]$Bucket,
        [Parameter(Mandatory = $true)][string]$Key
    )

    return "$script:McAlias/$Bucket/$Key"
}

function Test-S3Object {
    param(
        [Parameter(Mandatory = $true)][string]$Bucket,
        [Parameter(Mandatory = $true)][string]$Key
    )

    & $script:McBin stat (Get-S3RemotePath -Bucket $Bucket -Key $Key) *> $null
    return $LASTEXITCODE -eq 0
}

function Upload-ToHetznerS3 {
    param(
        [Parameter(Mandatory = $true)][string]$LocalPath,
        [Parameter(Mandatory = $true)][string]$BlobName,
        [Parameter(Mandatory = $true)][string]$Container,
        [Parameter(Mandatory = $true)][bool]$Overwrite
    )

    $bucket = Get-S3BucketForContainer $Container
    $key = Get-S3KeyForBlob $BlobName
    Initialize-HetznerS3

    if (-not $Overwrite -and (Test-S3Object -Bucket $bucket -Key $key)) {
        Write-Host "  [ok] Object $key already exists in bucket '$bucket' - skipping upload."
        return
    }

    if ($Overwrite) {
        Write-Host "  [upload] Uploading $key to bucket '$bucket' with overwrite enabled..."
    } else {
        Write-Host "  [upload] Uploading $key to bucket '$bucket'..."
    }

    & $script:McBin --quiet cp $LocalPath (Get-S3RemotePath -Bucket $bucket -Key $key)
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "Failed to upload $key to bucket '$bucket'."
    }
    Write-Host "  [ok] Uploaded $key."
}

function Upload-ToAzure {
    param(
        [Parameter(Mandatory = $true)][string]$LocalPath,
        [Parameter(Mandatory = $true)][string]$BlobName,
        [Parameter(Mandatory = $true)][string]$Container,
        [Parameter(Mandatory = $true)][bool]$Overwrite
    )

    $storageAcct = Get-AzureStorageAccount

    if ($Overwrite) {
        Write-Host "  [upload] Uploading $BlobName to container '$Container' with overwrite enabled..."
        $uploadArgs = @('storage', 'blob', 'upload', '--account-name', $storageAcct, '--container-name', $Container, '--name', $BlobName, '--file', $LocalPath, '--auth-mode', 'login', '--overwrite', 'true')
        az @uploadArgs
        if ($LASTEXITCODE -ne 0) {
            Write-Fail "Failed to upload $BlobName to Azure container '$Container'."
        }
        Write-Host "  [ok] Uploaded $BlobName."
    } else {
        $existsArgs = @('storage', 'blob', 'exists', '--account-name', $storageAcct, '--container-name', $Container, '--name', $BlobName, '--auth-mode', 'login', '--query', 'exists', '--output', 'tsv')
        $blobExists = az @existsArgs

        if ($blobExists -eq "true") {
            Write-Host "  [ok] Blob $BlobName already exists in '$Container' - skipping upload."
        } else {
            Write-Host "  [upload] Uploading $BlobName to container '$Container'..."
            $uploadArgs = @('storage', 'blob', 'upload', '--account-name', $storageAcct, '--container-name', $Container, '--name', $BlobName, '--file', $LocalPath, '--auth-mode', 'login', '--overwrite', 'false')
            az @uploadArgs
            if ($LASTEXITCODE -ne 0) {
                Write-Fail "Failed to upload $BlobName to Azure container '$Container'."
            }
            Write-Host "  [ok] Uploaded $BlobName."
        }
    }
}

foreach ($f in $files) {
    $localPath = $f.'local-path'
    $blobName  = $f.'blob-name'
    if (-not $blobName) { $blobName = Split-Path -Leaf $localPath }
    $container = $f.'container'
    if (-not $container) { $container = "build-cache" }
    $overwrite = $false
    if ($null -ne $f.PSObject.Properties['overwrite']) {
        $overwrite = [System.Convert]::ToBoolean($f.'overwrite')
    }

    Write-Host "=== Build cache upload: $localPath [object=$blobName container=$container overwrite=$overwrite backend=$cacheBackend] ==="

    if (-not (Test-Path $localPath -PathType Leaf)) {
        Write-Host "  [miss] File not found at $localPath - skipping."
        continue
    }

    # 1. Save to local runner cache when available.
    if ($localCacheDir -and (Test-Path $localCacheDir -PathType Container)) {
        $cached = Join-Path $localCacheDir $blobName
        $cachedParent = Split-Path -Parent $cached
        if ($cachedParent -and -not (Test-Path $cachedParent -PathType Container)) {
            New-Item -ItemType Directory -Path $cachedParent -Force | Out-Null
        }

        if ((Test-Path $cached -PathType Leaf) -and -not $overwrite) {
            Write-Host "  [ok] Already in local runner cache at $cached."
        } else {
            if (Test-Path $cached -PathType Leaf) {
                Write-Host "  [cache] Replacing local runner cache entry: $cached"
            } else {
                Write-Host "  [cache] Saving to local runner cache: $cached"
            }
            Copy-Item -Path $localPath -Destination $cached -Force
            Write-Host "  [ok] Saved to local runner cache."
        }
    }

    if ($cacheBackend -eq 'hetzner-s3' -or $cacheBackend -eq 's3') {
        Upload-ToHetznerS3 -LocalPath $localPath -BlobName $blobName -Container $container -Overwrite $overwrite
    } elseif ($cacheBackend -eq 'azure' -or $cacheBackend -eq 'azure-blob') {
        Upload-ToAzure -LocalPath $localPath -BlobName $blobName -Container $container -Overwrite $overwrite
    } else {
        Write-Fail "Unsupported storage backend '$cacheBackend'. Use 'hetzner-s3' or 'azure'."
    }
}
