# Ansible Role Sources

Dev Alchemy can build the Ansible role search path and default playbook path
from a small YAML config file. This lets you keep the bundled example roles as
a fallback, layer private roles above public roles, or point provisioning at
roles and playbooks you are actively developing.

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

Set `playbook` when this source stack should use a different default playbook
than `./playbooks/setup.yml`. Relative playbook paths are resolved through the
configured `playbook_sources` first, then through the bundled Dev Alchemy
project when `include_default_playbooks` is enabled. The `--playbook` CLI flag
still wins when it is set.

```yaml
playbook: custom-setup.yml
include_default_roles: true
include_default_playbooks: true
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

playbook_sources:
  - name: private-playbooks
    type: local
    path: /Users/me/src/my-dev-playbooks
```

## Source types

Local sources use `path`. Relative paths are resolved from the config file
directory.

Git sources use `url`. Dev Alchemy clones them into
`DEV_ALCHEMY_APP_DATA_DIR/cache/ansible-role-sources/` for roles and
`DEV_ALCHEMY_APP_DATA_DIR/cache/ansible-playbook-sources/` for playbooks, then
updates existing checkouts before provisioning. Set `ref` for a branch, tag, or
commit.

Set `roles_path` when a source stores roles in a subdirectory. Set
`playbooks_path` when a playbook source stores playbooks in a subdirectory.
For local or Git projects that contain both roles and playbooks, put both paths
on the same `sources` entry:

```yaml
playbook: setup.yml
include_default_roles: false
include_default_playbooks: false
sources:
  - name: dev-stack
    type: git
    url: https://github.com/example/dev-stack.git
    ref: main
    roles_path: roles
    playbooks_path: playbooks
```

If `include_default_roles` is `false`, make sure `playbook` points at an
entrypoint whose role names exist in your configured sources, or provide all
roles expected by the bundled `./playbooks/setup.yml`.

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
playbook: ./playbooks/role-sources-test.yml
include_default_roles: false
include_default_playbooks: true
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
alchemy provision local --check
```
