# Example Ansible Roles

This project contains a growing set of example Ansible roles. Use this page as
the lightweight catalog and starting point, and use [`roles/`](../roles/) for
the implementation details.

Current example roles in the repository:

- `brew`: installs and bootstraps Homebrew on macOS
- `java`: installs Temurin/OpenJDK-based Java runtimes
- `jetbrains`: installs JetBrains Toolbox
- `k9s`: installs the Kubernetes terminal UI
- `kind`: installs Kubernetes in Docker (`kind`)
- `kubectl`: installs the Kubernetes CLI
- `kubelogin`: installs the Kubernetes/OpenID login helper
- `openssh`: configures OpenSSH support for Windows targets
- `python`: installs Python tooling
- `spotify`: installs Spotify where supported

## Repository structure

The role-oriented parts of the repository are organized like this:

```text
devalchemy/
├── roles/
│   ├── brew/
│   ├── java/
│   └── python/
├── inventory/
│   ├── localhost.yaml
│   └── ...
├── playbooks/
│   ├── setup.yml
│   └── ...
└── docs/
    └── example-roles.md
```

- Add or adjust role logic under [`roles/`](../roles/).
- Use inventories under [`inventory/`](../inventory/) for localhost, remote
  hosts, or Windows access methods.
- Run the shared entrypoint from [`playbooks/setup.yml`](../playbooks/setup.yml)
  unless a role-specific playbook is a better fit.

## Customization

Run only selected tagged tasks:

```bash
ansible-playbook playbooks/setup.yml -i inventory/localhost.yaml --tags "dotfiles,python"
```

Pass variables to enable or tune behavior:

```bash
ansible-playbook playbooks/setup.yml -i inventory/localhost.yaml -e "install_docker=true"
```

For broader localhost, remote-host, VM, and Windows command examples, see
[Running Playbooks](./running-playbooks.md).

The exact role set can evolve over time. For the source of truth, inspect the
directories under [`roles/`](../roles/).
