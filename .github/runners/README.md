# macOS Tart Self-Hosted Runner

This directory contains scripts to build and operate ephemeral macOS GitHub Actions
self-hosted runners using [Tart](https://tart.run) virtualisation.

## Scripts

| Script | Purpose |
|---|---|
| `prepare-tart-base.sh` | One-time: builds the golden Tart VM image with tooling pre-installed. |
| `create-macos-tart-runner.sh` | Ongoing: runs a pool of ephemeral runners, one job per VM clone. |

## Quick start

```bash
# 1. Build the golden image once
./prepare-tart-base.sh

# 2. Start the runner pool (loops forever)
GITHUB_REPO=myorg/myrepo ./create-macos-tart-runner.sh
```

---

## Windows ISO cache

Workflows that test Packer builds require a Windows 11 ISO (`win11_25h2_english_arm64.iso` or
`win11_25h2_english_amd64.iso`). These files are several gigabytes. On runners with a slow or
metered internet connection the Azure Blob Storage download dominates job run-time.

To avoid repeated downloads you can maintain a local ISO cache on the **host machine**. The runner
script mounts the directory into every VM at boot via VirtioFS (read-write). The workflow then:

1. **Cache hit** — symlinks the ISO from `/Volumes/My Shared Files/iso-cache/` into the workspace; the
   Azure download is skipped entirely.
2. **Cache miss** — downloads the ISO from Azure Blob Storage as normal, then **copies it into
   the cache** so every subsequent run on the same runner is a cache hit.

No manual preparation of individual runners is needed. Point all of them at an empty directory
and the first job will populate it automatically.

### How it works

```
First run (cache empty)
──────────────────────────────────────────────────────────────
Azure Blob Storage
    │  download-windows-iso action
    ▼
Workflow workspace: cache/windows11/iso/win11_25h2_english_arm64.iso  (real file)
    │  "Save Windows ISO to local runner cache" step  (cp)
    ▼
Host machine: ~/iso-cache/win11_25h2_english_arm64.iso           (persists)

Every subsequent run (cache warm)
──────────────────────────────────────────────────────────────
Host machine: ~/iso-cache/win11_25h2_english_arm64.iso
    │  VirtioFS (tart --dir, read-write)
    ▼
Tart VM: /Volumes/My Shared Files/iso-cache/win11_25h2_english_arm64.iso
    │  "Link cached Windows ISO" step  (ln -sf)
    ▼
Workflow workspace: cache/windows11/iso/win11_25h2_english_arm64.iso  (symlink → no download)
```

### Set up the cache directory (one time per host)

Just create an empty directory — the first workflow run fills it:

```bash
mkdir -p ~/iso-cache
```

That's it. Alternatively, if you want to pre-seed the cache to avoid the first slow download
(e.g. you have the ISO on a USB drive or another machine):

```bash
# Option A – copy from USB drive / NAS / another machine:
cp /path/to/win11_25h2_english_arm64.iso ~/iso-cache/

# Option B – download once from Azure Blob Storage:
SUBSCRIPTION_ID="<your-subscription-id>"
STORAGE_ACCOUNT="ghrunner$(echo $SUBSCRIPTION_ID | tr -d '-' | cut -c1-24)"

az storage blob download \
  --account-name "$STORAGE_ACCOUNT" \
  --container-name windows-isos \
  --name win11_25h2_english_arm64.iso \
  --file ~/iso-cache/win11_25h2_english_arm64.iso \
  --auth-mode login
```

### Start the runner with caching enabled

Pass the `ISO_CACHE_DIR` environment variable when launching the runner script:

```bash
ISO_CACHE_DIR=~/iso-cache \
GITHUB_REPO=myorg/myrepo \
./create-macos-tart-runner.sh

ISO_CACHE_DIR=~/iso-cache \
RUNNER_POOL_SIZE=3 VM_CPU_COUNT=4 VM_MEMORY_MB=8192 \
./create-macos-tart-runner.sh
```

The directory is mounted read-write inside every VM as `/Volumes/iso-cache/`.

### Keeping the ISO up to date

Windows ISOs are large and rarely change. When a new version appears:

1. Delete (or rename) the old file in `~/iso-cache/` on each host machine.
2. The next workflow run will automatically download the new ISO from Azure Blob Storage and
   repopulate the cache.
3. Alternatively, copy the new ISO directly into `~/iso-cache/` to avoid any slow first run.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `ISO_CACHE_DIR` | _(unset)_ | Absolute path on the **host** to the ISO cache directory. When unset, no directory is mounted and jobs download ISOs normally. |
| `VM_BASE_IMAGE` | `tahoe-runner` | Tart image used as the base for each ephemeral clone. |
| `VM_CLONE_PER_RUN` | `true` | Clone a fresh VM for every runner cycle (recommended). |
| `GITHUB_REPO` | `csautter/dev-alchemy` | `owner/repo` for runner registration. |
| `GITHUB_SCOPE` | `repo` | `repo` or `org`. |
| `RUNNER_LABELS` | `macos,tart,arm64,macos-26-tart` | Comma-separated runner labels. |
| `MAX_RUNS` | `0` (infinite) | Stop the loop after this many runner cycles. |
