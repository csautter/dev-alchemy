#!/usr/bin/env bash
# Expects the following environment variables (set by the composite action):
#   CACHE_FILES       - JSON array of cache file descriptors
#   SUBSCRIPTION_ID   - Azure subscription ID
#   LOCAL_CACHE_DIR   - Path to the local runner shared-volume cache (may be empty)
set -euo pipefail

STORAGE_ACCOUNT=$(echo "ghrunner${SUBSCRIPTION_ID}" | tr -cd '[:alnum:]' | tr '[:upper:]' '[:lower:]' | cut -c1-24)

echo "$CACHE_FILES" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for f in data:
    print(f.get('local-path', ''))
    print(f.get('blob-name', ''))
    print(f.get('container', 'build-cache'))
    print(str(f.get('overwrite', False)).lower())
" | while IFS= read -r local_path && IFS= read -r blob_name && IFS= read -r container && IFS= read -r overwrite; do
  [ -z "$local_path" ] && continue
  [ -z "$blob_name" ] && blob_name=$(basename "$local_path")
  [ -z "$overwrite" ] && overwrite=false

  echo "=== Build cache upload: $local_path [blob=$blob_name container=$container overwrite=$overwrite] ==="

  # Resolve symlinks so we operate on the real bytes
  real_path="$local_path"
  if [ -L "$local_path" ]; then
    real_path=$(readlink -f "$local_path")
  fi

  if [ ! -f "$real_path" ]; then
    echo "  ✗ File not found at $local_path — skipping."
    continue
  fi

  # 1. Save to local runner shared-volume cache (macOS Tart)
  if [ -n "$LOCAL_CACHE_DIR" ] && [ -d "$LOCAL_CACHE_DIR" ]; then
    cached="${LOCAL_CACHE_DIR}/${blob_name}"
    if [ -L "$local_path" ]; then
      echo "  ✓ File is a symlink to local cache — already cached."
    elif [ -f "$cached" ] && [ "$overwrite" != "true" ]; then
      echo "  ✓ Already in local runner cache at $cached."
    else
      mkdir -p "$(dirname "$cached")"
      if [ -f "$cached" ] && [ "$overwrite" = "true" ]; then
        echo "  ↻ Replacing local runner cache entry at $cached"
      else
        echo "  ↑ Saving to local runner cache: $cached"
      fi
      cp "$real_path" "$cached"
      echo "  ✓ Saved to local runner cache."
    fi
  fi

  # 2. Upload to Azure Blob Storage
  if [ "$overwrite" = "true" ]; then
    echo "  ↻ Uploading $blob_name to container '$container' with overwrite enabled..."
    az storage blob upload \
      --account-name "$STORAGE_ACCOUNT" \
      --container-name "$container" \
      --name "$blob_name" \
      --file "$real_path" \
      --auth-mode login \
      --overwrite true
    echo "  ✓ Uploaded $blob_name."
  else
    blob_exists=$(az storage blob exists \
      --account-name "$STORAGE_ACCOUNT" \
      --container-name "$container" \
      --name "$blob_name" \
      --auth-mode login \
      --query "exists" \
      --output tsv)

    if [ "$blob_exists" = "true" ]; then
      echo "  ✓ Blob $blob_name already exists in '$container' — skipping upload."
    else
      echo "  ↑ Uploading $blob_name to container '$container'..."
      az storage blob upload \
        --account-name "$STORAGE_ACCOUNT" \
        --container-name "$container" \
        --name "$blob_name" \
        --file "$real_path" \
        --auth-mode login \
        --overwrite false
      echo "  ✓ Uploaded $blob_name."
    fi
  fi
done
