# Ansible Role Sources

Dev Alchemy can build the Ansible role search path from a small YAML config
file. This lets you keep the bundled example roles as a fallback, layer private
roles above public roles, or point provisioning at roles you are actively
developing.

## Config location

By default, Dev Alchemy reads `ansible-role-sources.yml` from the OS-specific
config directory:

- macOS: `~/Library/Application Support/dev-alchemy/ansible-role-sources.yml`
- Windows: `%APPDATA%\dev-alchemy\ansible-role-sources.yml`
- Linux: `${XDG_CONFIG_HOME:-~/.config}/dev-alchemy/ansible-role-sources.yml`

Set `DEV_ALCHEMY_CONFIG_DIR` to move the config directory, or set
`DEV_ALCHEMY_ROLE_SOURCES_CONFIG` to point at one specific config file.

## Layering model

Sources are applied in the order they appear in the file. Ansible uses the
first matching role name it finds, so put specific override roles before base
role collections.

If the config file is missing, Dev Alchemy keeps the old behavior and uses the
bundled `./roles` directory. When the config file exists,
`include_default_roles` defaults to `true`, so configured sources are placed
before the bundled roles.

```yaml
include_default_roles: true
sources:
  - name: private-overrides
    type: local
    path: /Users/me/src/my-dev-roles

  - name: public-base
    type: git
    url: https://github.com/csautter/dev-alchemy.git
    ref: main
    roles_path: roles
    update: pull
```

## Source types

Local sources use `path`. Relative paths are resolved from the config file
directory.

Git sources use `url`. Dev Alchemy clones them into
`DEV_ALCHEMY_APP_DATA_DIR/cache/ansible-role-sources/` and updates existing
checkouts before provisioning. Set `ref` for a branch, tag, or commit. Set
`roles_path` when the repository stores roles in a subdirectory.

Update behavior can be disabled for a Git source:

```yaml
sources:
  - name: pinned-base
    type: git
    url: https://github.com/csautter/dev-alchemy.git
    ref: v1.0.0
    pull: false
```

## Manual Layer Test

The repository includes two tiny role-source folders for hand testing:
`roles_test_1/` and `roles_test_2/`. Put the overlay first to verify that the
shared role resolves from `roles_test_2`.

```yaml
include_default_roles: true
sources:
  - name: test-overlay
    type: local
    path: /path/to/dev-alchemy/roles_test_2
  - name: test-base
    type: local
    path: /path/to/dev-alchemy/roles_test_1
```

Then run:

```bash
alchemy provision local --playbook ./playbooks/role-sources-test.yml --check
```
