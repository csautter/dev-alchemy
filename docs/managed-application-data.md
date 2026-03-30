# Managed Application Data

Dev Alchemy stores VM build state, deployment state, caches, and standalone
runtime assets outside the repository in an OS-appropriate app-data directory.

## Default location

- macOS: `~/Library/Application Support/dev-alchemy`
- Windows: `%LOCALAPPDATA%\dev-alchemy`
- Linux: `${XDG_DATA_HOME:-~/.local/share}/dev-alchemy`

## Managed subdirectories

Under that root, Dev Alchemy manages:

- `cache/` for downloaded files and build artifacts
- `.vagrant/` for isolated Vagrant state
- `packer_cache/` for Packer plugin and download cache
- `project/` for the embedded runtime project used by standalone binaries
  outside a Git checkout

## Overrides and exported paths

You can override the default root by setting
`DEV_ALCHEMY_APP_DATA_DIR`.

Dev Alchemy also exports these derived paths for helper scripts and manual
workflows:

- `DEV_ALCHEMY_CACHE_DIR`
- `DEV_ALCHEMY_VAGRANT_DIR`
- `DEV_ALCHEMY_PACKER_CACHE_DIR`

## Standalone runtime assets

On the first standalone run, Dev Alchemy extracts bundled scripts, playbooks,
and other runtime assets into `DEV_ALCHEMY_APP_DATA_DIR/project`.

Later runs keep that managed tree in sync so the standalone `alchemy` binary
can operate without a repository checkout.
