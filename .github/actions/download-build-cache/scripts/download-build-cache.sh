#!/usr/bin/env bash
# Expects the following environment variables (set by the composite action):
#   CACHE_FILES       - JSON array of cache file descriptors
#   SUBSCRIPTION_ID   - Azure subscription ID
#   LOCAL_CACHE_DIR   - Path to the local runner shared-volume cache (may be empty)
set -euo pipefail

STORAGE_ACCOUNT=$(echo "ghrunner${SUBSCRIPTION_ID}" | tr -cd '[:alnum:]' | tr '[:upper:]' '[:lower:]' | cut -c1-24)
RESOURCE_GROUP="gh-runner-storage-rg"

# Iterate over each entry in the JSON array using Python (always available on runners)
echo "$CACHE_FILES" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for f in data:
    print(f.get('local-path', ''))
    print(f.get('blob-name', ''))
    print(f.get('container', 'build-cache'))
" | while IFS= read -r local_path && IFS= read -r blob_name && IFS= read -r container; do
  [ -z "$local_path" ] && continue
  [ -z "$blob_name" ] && blob_name=$(basename "$local_path")

  echo "=== Build cache: $local_path [blob=$blob_name container=$container] ==="

  # 1. Already present in workspace → nothing to do
  if [ -f "$local_path" ]; then
    echo "  ✓ Already present at $local_path — skipping."
    continue
  fi

  # 2. Local runner shared-volume cache (macOS Tart VirtioFS) → symlink
  if [ -n "$LOCAL_CACHE_DIR" ] && [ -f "${LOCAL_CACHE_DIR}/${blob_name}" ]; then
    echo "  ✓ Found in local runner cache — creating symlink."
    mkdir -p "$(dirname "$local_path")"
    ln -sf "${LOCAL_CACHE_DIR}/${blob_name}" "$local_path"
    continue
  fi

  # 3. Azure Blob Storage → download
  echo "  ↓ Not in local cache. Attempting Azure Blob Storage download..."
  if az storage account show \
      --name "$STORAGE_ACCOUNT" \
      --resource-group "$RESOURCE_GROUP" &>/dev/null; then
    blob_exists=$(az storage blob exists \
      --account-name "$STORAGE_ACCOUNT" \
      --container-name "$container" \
      --name "$blob_name" \
      --auth-mode login \
      --query "exists" \
      --output tsv)
    if [ "$blob_exists" = "true" ]; then
      mkdir -p "$(dirname "$local_path")"
      az storage blob download \
        --account-name "$STORAGE_ACCOUNT" \
        --container-name "$container" \
        --name "$blob_name" \
        --file "$local_path" \
        --auth-mode login
      echo "  ✓ Downloaded $blob_name → $local_path"

      # Save to local runner shared-volume cache (macOS Tart)
      if [ -n "$LOCAL_CACHE_DIR" ] && [ -d "$LOCAL_CACHE_DIR" ]; then
        cached="${LOCAL_CACHE_DIR}/${blob_name}"
        if [ -f "$cached" ]; then
          echo "  ✓ Already in local runner cache at $cached."
        else
          echo "  ↑ Saving to local runner cache: $cached"
          cp "$local_path" "$cached"
          echo "  ✓ Saved to local runner cache."
        fi
      fi
    else
      echo "  ✗ Blob $blob_name not found in container '$container'."
    fi
  else
    echo "  ✗ Storage account $STORAGE_ACCOUNT not found."
  fi
done
