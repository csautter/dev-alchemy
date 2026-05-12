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
$resourceGroup  = 'gh-runner-storage-rg'
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

function Save-ToLocalCache {
    param(
        [Parameter(Mandatory = $true)][string]$LocalPath,
        [Parameter(Mandatory = $true)][string]$BlobName
    )

    if ($localCacheDir -and (Test-Path $localCacheDir -PathType Container)) {
        $cached = Join-Path $localCacheDir $BlobName
        if (Test-Path $cached -PathType Leaf) {
            Write-Host "  [ok] Already in local runner cache at $cached."
        } else {
            $cachedParent = Split-Path -Parent $cached
            if ($cachedParent -and -not (Test-Path $cachedParent -PathType Container)) {
                New-Item -ItemType Directory -Path $cachedParent -Force | Out-Null
            }
            Write-Host "  [cache] Saving to local runner cache: $cached"
            Copy-Item -Path $LocalPath -Destination $cached -Force
            Write-Host '  [ok] Saved to local runner cache.'
        }
    }
}

function Download-FromHetznerS3 {
    param(
        [Parameter(Mandatory = $true)][string]$LocalPath,
        [Parameter(Mandatory = $true)][string]$BlobName,
        [Parameter(Mandatory = $true)][string]$Container
    )

    $bucket = Get-S3BucketForContainer $Container
    $key = Get-S3KeyForBlob $BlobName

    Write-Host "  [download] Not in local cache. Attempting Hetzner S3 download from bucket '$bucket' key '$key'..."
    Initialize-HetznerS3

    if (Test-S3Object -Bucket $bucket -Key $key) {
        $dir = Split-Path -Path $LocalPath -Parent
        if ($dir -and -not (Test-Path $dir -PathType Container)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }

        & $script:McBin --quiet cp (Get-S3RemotePath -Bucket $bucket -Key $key) $LocalPath
        if ($LASTEXITCODE -ne 0) {
            Write-Fail "Failed to download $key from bucket '$bucket'."
        }
        Write-Host "  [ok] Downloaded $key -> $LocalPath"
        Save-ToLocalCache -LocalPath $LocalPath -BlobName $BlobName
    } else {
        Write-Host "  [miss] Object $key not found in bucket '$bucket'."
    }
}

function Download-FromAzure {
    param(
        [Parameter(Mandatory = $true)][string]$LocalPath,
        [Parameter(Mandatory = $true)][string]$BlobName,
        [Parameter(Mandatory = $true)][string]$Container
    )

    $storageAcct = Get-AzureStorageAccount

    Write-Host '  [download] Attempting Azure Blob Storage download...'
    $saArgs = @('storage', 'account', 'show', '--name', $storageAcct, '--resource-group', $resourceGroup)
    $saJson = az @saArgs 2>&1
    $sa = $null
    if ($LASTEXITCODE -eq 0) { $sa = $saJson | ConvertFrom-Json }

    if ($sa) {
        $existsArgs = @('storage', 'blob', 'exists', '--account-name', $storageAcct, '--container-name', $Container, '--name', $BlobName, '--auth-mode', 'login')
        $blobExists = (az @existsArgs | ConvertFrom-Json).exists

        if ($blobExists) {
            $dir = Split-Path -Path $LocalPath -Parent
            if ($dir -and -not (Test-Path $dir -PathType Container)) {
                New-Item -ItemType Directory -Path $dir -Force | Out-Null
            }
            $dlArgs = @('storage', 'blob', 'download', '--account-name', $storageAcct, '--container-name', $Container, '--name', $BlobName, '--file', $LocalPath, '--auth-mode', 'login')
            az @dlArgs
            if ($LASTEXITCODE -ne 0) {
                Write-Fail "Failed to download $BlobName from Azure container '$Container'."
            }
            Write-Host "  [ok] Downloaded $BlobName -> $LocalPath"
            Save-ToLocalCache -LocalPath $LocalPath -BlobName $BlobName
        } else {
            Write-Host "  [miss] Blob $BlobName not found in container '$Container'."
        }
    } else {
        Write-Host "  [miss] Storage account $storageAcct not found."
    }
}

foreach ($f in $files) {
    $localPath = $f.'local-path'
    $blobName  = $f.'blob-name'
    if (-not $blobName) { $blobName = Split-Path -Leaf $localPath }
    $container = $f.'container'
    if (-not $container) { $container = "build-cache" }

    Write-Host "=== Build cache: $localPath [object=$blobName container=$container backend=$cacheBackend] ==="

    # 1. Already present in workspace -> nothing to do
    if (Test-Path $localPath) {
        Write-Host "  [ok] Already present at $localPath - skipping."
        continue
    }

    # 2. Local runner cache -> copy
    if ($localCacheDir -and (Test-Path $localCacheDir -PathType Container)) {
        $cached = Join-Path $localCacheDir $blobName
        if (Test-Path $cached -PathType Leaf) {
            Write-Host '  [ok] Found in local runner cache - copying.'
            $dir = Split-Path -Path $localPath -Parent
            if ($dir -and -not (Test-Path $dir -PathType Container)) {
                New-Item -ItemType Directory -Path $dir -Force | Out-Null
            }
            Copy-Item -Path $cached -Destination $localPath -Force
            continue
        }
    }

    if ($cacheBackend -eq 'hetzner-s3' -or $cacheBackend -eq 's3') {
        Download-FromHetznerS3 -LocalPath $localPath -BlobName $blobName -Container $container
    } elseif ($cacheBackend -eq 'azure' -or $cacheBackend -eq 'azure-blob') {
        Download-FromAzure -LocalPath $localPath -BlobName $blobName -Container $container
    } else {
        Write-Fail "Unsupported storage backend '$cacheBackend'. Use 'hetzner-s3' or 'azure'."
    }
}
