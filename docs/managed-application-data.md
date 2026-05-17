# Managed Application Data

Dev Alchemy stores VM build state, deployment state, caches, and standalone
runtime assets outside the repository in an OS-appropriate app-data directory.
User-editable configuration is stored in an OS-appropriate config directory.

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

## Config location

Dev Alchemy reads user-editable config from:

- macOS: `~/Library/Application Support/dev-alchemy`
- Windows: `%APPDATA%\dev-alchemy`
- Linux: `${XDG_CONFIG_HOME:-~/.config}/dev-alchemy`

The Ansible role-source config is
`ansible-role-sources.yml` in that directory. See
[Ansible Role Sources](./ansible-role-sources.md).

## Overrides and exported paths

You can override the default root by setting
`DEV_ALCHEMY_APP_DATA_DIR`.
You can override the config directory by setting
`DEV_ALCHEMY_CONFIG_DIR`.

Dev Alchemy also exports these derived paths for helper scripts and manual
workflows:

- `DEV_ALCHEMY_CONFIG_DIR`
- `DEV_ALCHEMY_CACHE_DIR`
- `DEV_ALCHEMY_VAGRANT_DIR`
- `DEV_ALCHEMY_PACKER_CACHE_DIR`

## Standalone runtime assets

On the first standalone run, Dev Alchemy extracts bundled scripts, playbooks,
and other runtime assets into `DEV_ALCHEMY_APP_DATA_DIR/project`.

Later runs keep that managed tree in sync so the standalone `alchemy` binary
can operate without a repository checkout.
