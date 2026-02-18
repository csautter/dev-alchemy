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
script will mount the directory into every VM at boot via VirtioFS, and the workflow job will
symlink the ISO from the mount-point instead of downloading it.

### How it works

```
Host machine
└── ~/iso-cache/
    ├── win11_25h2_english_arm64.iso   ← lives here, never copied
    └── win11_25h2_english_amd64.iso

      │  VirtioFS (tart --dir flag)
      ▼

Tart VM
└── /Volumes/iso-cache/
    ├── win11_25h2_english_arm64.iso   ← read-only view
    └── win11_25h2_english_amd64.iso

      │  ln -sf
      ▼

Workflow workspace
└── vendor/windows/win11_25h2_english_arm64.iso  ← symlink only
```

The download-windows-iso action checks whether the file already exists before contacting Azure Blob
Storage, so the symlink is sufficient to skip the download entirely.

### Set up the cache (one time)

```bash
# Create the cache directory
mkdir -p ~/iso-cache

# Download the ISO(s) you need.
# Option A – from Azure Blob Storage (requires az CLI login):
SUBSCRIPTION_ID="<your-subscription-id>"
STORAGE_ACCOUNT="ghrunner$(echo $SUBSCRIPTION_ID | tr -d '-' | cut -c1-24)"

az storage blob download \
  --account-name "$STORAGE_ACCOUNT" \
  --container-name windows-isos \
  --name win11_25h2_english_arm64.iso \
  --file ~/iso-cache/win11_25h2_english_arm64.iso \
  --auth-mode login

# Option B – copy from another machine, USB drive, or NAS:
cp /path/to/win11_25h2_english_arm64.iso ~/iso-cache/
```

### Start the runner with caching enabled

Pass the `ISO_CACHE_DIR` environment variable when launching the runner script:

```bash
ISO_CACHE_DIR=~/iso-cache \
GITHUB_REPO=myorg/myrepo \
./create-macos-tart-runner.sh
```

The directory is mounted read-only inside every VM as `/Volumes/iso-cache/`. The workflow's
"Link cached Windows ISO" step automatically creates the symlink; if the file is absent the job
falls back to the normal Azure Blob Storage download without any manual intervention.

### Keeping the ISO up to date

Windows ISOs are large and rarely change. A simple maintenance workflow:

1. When a new ISO version is added to Azure Blob Storage, re-download it to `~/iso-cache/` on
   each host machine.
2. The old ISO file can be deleted once no workflows reference the old filename.
3. If the host machine itself is recreated, re-run the one-time setup steps above.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `ISO_CACHE_DIR` | _(unset)_ | Absolute path on the **host** to the ISO cache directory. When unset, no directory is mounted and jobs download ISOs normally. |
| `VM_BASE_IMAGE` | `tahoe-runner` | Tart image used as the base for each ephemeral clone. |
| `VM_CLONE_PER_RUN` | `true` | Clone a fresh VM for every runner cycle (recommended). |
| `GITHUB_REPO` | `csautter/dev-alchemy` | `owner/repo` for runner registration. |
| `GITHUB_SCOPE` | `repo` | `repo` or `org`. |
| `RUNNER_LABELS` | `macos,tart,arm64,macos-16-tart` | Comma-separated runner labels. |
| `MAX_RUNS` | `0` (infinite) | Stop the loop after this many runner cycles. |
