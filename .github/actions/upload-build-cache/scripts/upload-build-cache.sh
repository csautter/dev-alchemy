#!/usr/bin/env bash
# Expects the following environment variables (set by the composite action):
#   CACHE_FILES       - JSON array of cache file descriptors
#   CACHE_BACKEND     - Remote cache backend: hetzner-s3 or azure
#   SUBSCRIPTION_ID   - Azure subscription ID (azure backend only)
#   HETZNER_S3_*      - Hetzner S3 endpoint, bucket, access key, secret key, and optional prefix
#   LOCAL_CACHE_DIR   - Path to the local runner shared-volume cache (may be empty)
set -euo pipefail

CACHE_BACKEND=$(printf '%s' "${CACHE_BACKEND:-hetzner-s3}" | tr '[:upper:]' '[:lower:]')
MC_ALIAS="dev-alchemy-build-cache"
MC_BIN=""
MC_READY=false

fail() {
  echo "  ✗ $*" >&2
  exit 1
}

normalize_endpoint_url() {
  local endpoint="$1"
  case "$endpoint" in
    http://*|https://*)
      printf '%s' "$endpoint"
      ;;
    *)
      printf 'https://%s' "$endpoint"
      ;;
  esac
}

derive_azure_storage_account() {
  if [ -z "${SUBSCRIPTION_ID:-}" ]; then
    fail "subscription-id is required when storage-backend is 'azure'."
  fi

  echo "ghrunner${SUBSCRIPTION_ID}" |
    tr -cd '[:alnum:]' |
    tr '[:upper:]' '[:lower:]' |
    cut -c1-24
}

ensure_mc() {
  if [ -n "$MC_BIN" ]; then
    return
  fi

  if command -v mc >/dev/null 2>&1; then
    MC_BIN="$(command -v mc)"
    return
  fi

  local os
  local arch
  case "$(uname -s)" in
    Linux)
      os="linux"
      ;;
    Darwin)
      os="darwin"
      ;;
    *)
      fail "Hetzner S3 backend requires the MinIO client, and automatic install is only supported on Linux and macOS."
      ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64)
      arch="amd64"
      ;;
    arm64|aarch64)
      arch="arm64"
      ;;
    *)
      fail "Unsupported architecture for MinIO client: $(uname -m)."
      ;;
  esac

  local install_dir="${RUNNER_TEMP:-/tmp}/dev-alchemy-build-cache"
  mkdir -p "$install_dir"
  MC_BIN="${install_dir}/mc"

  if [ ! -x "$MC_BIN" ]; then
    echo "  → Installing MinIO client for Hetzner S3 cache operations..."
    curl -fsSL -o "$MC_BIN" "https://dl.min.io/client/mc/release/${os}-${arch}/mc"
    chmod +x "$MC_BIN"
  fi
}

initialize_hetzner_s3() {
  if [ "$MC_READY" = "true" ]; then
    return
  fi

  local missing=()
  [ -n "${HETZNER_S3_ENDPOINT_URL:-}" ] || missing+=("HETZNER_S3_ENDPOINT_URL")
  [ -n "${HETZNER_S3_ACCESS_KEY_ID:-}" ] || missing+=("HETZNER_S3_ACCESS_KEY_ID")
  [ -n "${HETZNER_S3_SECRET_ACCESS_KEY:-}" ] || missing+=("HETZNER_S3_SECRET_ACCESS_KEY")
  if [ "${#missing[@]}" -gt 0 ]; then
    fail "Missing Hetzner S3 configuration: ${missing[*]}."
  fi

  ensure_mc

  export MC_CONFIG_DIR="${MC_CONFIG_DIR:-${RUNNER_TEMP:-/tmp}/dev-alchemy-build-cache-mc}"
  mkdir -p "$MC_CONFIG_DIR"

  local endpoint
  endpoint="$(normalize_endpoint_url "$HETZNER_S3_ENDPOINT_URL")"
  "$MC_BIN" alias set "$MC_ALIAS" "$endpoint" "$HETZNER_S3_ACCESS_KEY_ID" "$HETZNER_S3_SECRET_ACCESS_KEY" --api S3v4 --path auto >/dev/null
  MC_READY=true
}

s3_bucket_for_container() {
  local container="$1"
  if [ -n "${HETZNER_S3_BUCKET:-}" ]; then
    printf '%s' "$HETZNER_S3_BUCKET"
  else
    printf '%s' "$container"
  fi
}

s3_key_for_blob() {
  local blob_name="${1#/}"
  local prefix="${HETZNER_S3_PREFIX:-}"
  prefix="${prefix#/}"
  prefix="${prefix%/}"

  if [ -n "$prefix" ]; then
    printf '%s/%s' "$prefix" "$blob_name"
  else
    printf '%s' "$blob_name"
  fi
}

s3_remote_path() {
  local bucket="$1"
  local key="$2"
  printf '%s/%s/%s' "$MC_ALIAS" "$bucket" "$key"
}

s3_object_exists() {
  local bucket="$1"
  local key="$2"
  "$MC_BIN" stat "$(s3_remote_path "$bucket" "$key")" >/dev/null 2>&1
}

upload_to_hetzner_s3() {
  local real_path="$1"
  local blob_name="$2"
  local container="$3"
  local overwrite="$4"
  local bucket
  local key

  bucket="$(s3_bucket_for_container "$container")"
  key="$(s3_key_for_blob "$blob_name")"
  initialize_hetzner_s3

  if [ "$overwrite" != "true" ] && s3_object_exists "$bucket" "$key"; then
    echo "  ✓ Object $key already exists in bucket '$bucket' — skipping upload."
    return
  fi

  if [ "$overwrite" = "true" ]; then
    echo "  ↻ Uploading $key to bucket '$bucket' with overwrite enabled..."
  else
    echo "  ↑ Uploading $key to bucket '$bucket'..."
  fi
  "$MC_BIN" --quiet cp "$real_path" "$(s3_remote_path "$bucket" "$key")"
  echo "  ✓ Uploaded $key."
}

upload_to_azure() {
  local real_path="$1"
  local blob_name="$2"
  local container="$3"
  local overwrite="$4"
  local storage_account
  local blob_exists

  storage_account="$(derive_azure_storage_account)"

  if [ "$overwrite" = "true" ]; then
    echo "  ↻ Uploading $blob_name to container '$container' with overwrite enabled..."
    az storage blob upload \
      --account-name "$storage_account" \
      --container-name "$container" \
      --name "$blob_name" \
      --file "$real_path" \
      --auth-mode login \
      --overwrite true
    echo "  ✓ Uploaded $blob_name."
  else
    blob_exists=$(az storage blob exists \
      --account-name "$storage_account" \
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
        --account-name "$storage_account" \
        --container-name "$container" \
        --name "$blob_name" \
        --file "$real_path" \
        --auth-mode login \
        --overwrite false
      echo "  ✓ Uploaded $blob_name."
    fi
  fi
}

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
  [ -z "$container" ] && container="build-cache"
  [ -z "$overwrite" ] && overwrite=false

  echo "=== Build cache upload: $local_path [object=$blob_name container=$container overwrite=$overwrite backend=$CACHE_BACKEND] ==="

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

  case "$CACHE_BACKEND" in
    hetzner-s3|s3)
      upload_to_hetzner_s3 "$real_path" "$blob_name" "$container" "$overwrite"
      ;;
    azure|azure-blob)
      upload_to_azure "$real_path" "$blob_name" "$container" "$overwrite"
      ;;
    *)
      fail "Unsupported storage backend '$CACHE_BACKEND'. Use 'hetzner-s3' or 'azure'."
      ;;
  esac
done
