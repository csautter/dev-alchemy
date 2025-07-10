# 🧪 devalchemy

**devalchemy** is a cross-platform development environment automation toolkit powered by [Ansible](https://www.ansible.com/). It turns fresh machines into fully-configured dev setups — whether you're on **macOS**, **Linux**, or **Windows** (via WSL).

> _"Transform your system into a dev powerhouse — with a touch of automation magic."_

---

## ✨ Features

- ✅ Unified setup for macOS, Linux, and Windows (WSL)
- 📦 Install development tools, CLIs, languages, and more
- ⚙️ Easily extensible Ansible roles and playbooks
- 💻 Consistent dev experience across platforms
- 🔒 Minimal privileges needed (no full root where not required)

---

## 🚀 Getting Started

### 1. Clone the repo

```bash
git clone https://github.com/csautter/dev-alchemy.git
cd dev-alchemy

### 2. Install Ansible

> Make sure Ansible is installed on your system.

#### macOS (via Homebrew):

```bash
brew install ansible
```

#### Ubuntu / Debian:

```bash
sudo apt update && sudo apt install ansible
```

#### Windows (WSL):

```bash
sudo apt update && sudo apt install ansible
```

> ⚠️ Native Windows support is limited — use WSL for best results.

---

### 3. Run the Playbook

```bash
ansible-playbook setup.yml -i inventory/localhost.yml
```

You can customize the inventory or pass variables via CLI.

---

## 🧩 Structure

```
devalchemy/
├── roles/
│   ├── common/
│   ├── devtools/
│   └── languages/
├── inventory/
│   └── localhost.yml
├── setup.yml
└── README.md
```

---

## 🛠️ Customization

* Add or tweak roles in `roles/`

* Use tags to run specific parts:

  ```bash
  ansible-playbook setup.yml --tags "dotfiles,python"
  ```

* Pass variables:

  ```bash
  ansible-playbook setup.yml -e "install_docker=true"
  ```

---

## 📦 Supported Tools

Out-of-the-box roles can install (depending on platform):

* Git, Docker, Python, Node.js, Go
* Zsh / Oh My Zsh, VS Code, tmux, fzf
* CLI utilities: `htop`, `jq`, `ripgrep`, etc.

> Full list in `roles/` and tagged tasks

---

## 🌍 Cross-Platform Notes

| Platform | Status      | Notes                             |
| -------- | ----------- | --------------------------------- |
| macOS    | ✅ Supported | via Homebrew                      |
| Linux    | ✅ Supported | tested on Ubuntu, Debian, Arch    |
| Windows  | ⚠️ WSL only | tested on WSL2 with Ubuntu/Debian |

---

## 🤝 Contributing

Contributions welcome! Feel free to:

* Add new roles (e.g., Rust, Java, etc.)
* Improve cross-platform support
* Fix bugs or enhance docs

---

## 📜 License

MIT License — see [LICENSE](LICENSE) file.

---

## 💡 Inspiration

This project was born from a need to simplify dev environment onboarding across multiple systems, without resorting to OS-specific scripts. With Ansible and a touch of Dev Alchemy, setup becomes reproducible and delightful.

---

## 🔗 Related Projects

* [geerlingguy/mac-dev-playbook](https://github.com/geerlingguy/mac-dev-playbook)
* [ansible/ansible-examples](https://github.com/ansible/ansible-examples)

---

🧪 *Happy hacking with `devalchemy`!*
