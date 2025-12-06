# nvs

**Neovim Version Switcher** – Install, switch, and manage multiple Neovim versions with ease 🚀

[![CI](https://github.com/y3owk1n/nvs/actions/workflows/ci.yml/badge.svg)](https://github.com/y3owk1n/nvs/actions/workflows/ci.yml)
[![GitHub release](https://img.shields.io/github/release/y3owk1n/nvs.svg)](https://github.com/y3owk1n/nvs/releases)
[![License](https://img.shields.io/github/license/y3owk1n/nvs.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/y3owk1n/nvs)](https://goreportcard.com/report/github.com/y3owk1n/nvs)

---

## Why nvs?

Managing multiple Neovim versions shouldn't be complicated. **nvs** makes it simple:

- ⚡ **Instant Switching** – Switch between versions in milliseconds
- 📌 **Per-Project Versions** – Pin versions with `.nvs-version` files
- 🔄 **Auto-Switch** – Automatically switch versions when changing directories
- 🌙 **Nightly Support** – First-class support for nightly builds with rollback capability
- 🔧 **Build from Source** – Install any commit directly from the Neovim repository
- 🔀 **Config Switching** – Toggle between multiple Neovim configurations
- 🩺 **Self-Diagnosing** – Built-in health checks with `nvs doctor`

```bash
$ nvs use stable
✓ Switched to Neovim stable

$ nvim -v
NVIM v0.10.4
Build type: Release
LuaJIT 2.1.1713484068

$ nvs use nightly
✓ Switched to Neovim nightly

$ nvim -v
NVIM v0.11.0-dev-1961+g7e2b75760f
Build type: RelWithDebInfo
LuaJIT 2.1.1741571767
```

---

## Quick Start

### 1. Install nvs

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/install.sh | bash
```

**Windows (PowerShell):**

```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/y3owk1n/nvs/main/install.ps1" -OutFile "install.ps1"; .\install.ps1
```

**Other methods:** [Homebrew](docs/INSTALLATION.md#method-2-homebrew-macoslinux) · [Nix](docs/INSTALLATION.md#method-4-nix-flakes) · [From Source](docs/INSTALLATION.md#method-5-build-from-source)

### 2. Set Up Your Shell

Add to your shell configuration (`~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish`):

```bash
# Bash/Zsh
eval "$(nvs env --source)"

# Fish
nvs env --source | source
```

### 3. Install and Use Neovim

```bash
nvs install stable    # Install stable release
nvs use stable        # Switch to stable
nvim --version        # Verify installation
```

---

## Command Reference

| Command                   | Description                                                            |
| ------------------------- | ---------------------------------------------------------------------- |
| `nvs install <version>`   | Install a Neovim version (`stable`, `nightly`, `v0.10.3`, commit hash) |
| `nvs use <version>`       | Switch to an installed version                                         |
| `nvs list`                | List installed versions                                                |
| `nvs list-remote`         | List available remote versions                                         |
| `nvs current`             | Show currently active version                                          |
| `nvs upgrade`             | Upgrade stable and/or nightly versions                                 |
| `nvs uninstall <version>` | Remove an installed version                                            |
| `nvs pin [version]`       | Pin version to current directory (`.nvs-version`)                      |
| `nvs rollback`            | Rollback to a previous nightly version                                 |
| `nvs run <version>`       | Run a version without switching                                        |
| `nvs config`              | Switch Neovim configuration                                            |
| `nvs doctor`              | Check system health                                                    |
| `nvs hook <shell>`        | Generate shell hook for auto-switching                                 |

See the [Usage Guide](docs/USAGE.md) for detailed examples and options.

---

## Features

### Version Management

Install any Neovim version – stable releases, nightly builds, specific tags, or even build from any commit:

```bash
nvs install stable           # Latest stable
nvs install nightly          # Latest nightly
nvs install v0.10.3          # Specific version
nvs install 2db1ae3          # Build from commit
```

### Per-Project Version Pinning

Pin a Neovim version to your project directory:

```bash
nvs pin stable               # Creates .nvs-version
nvs use                      # Reads from .nvs-version
```

Enable auto-switching to automatically switch versions when entering a directory:

```bash
# Add to shell config
eval "$(nvs hook bash)"      # or zsh/fish
```

### Nightly Management

Keep multiple nightly versions and rollback when needed:

```bash
nvs upgrade nightly          # Upgrade to latest nightly
nvs rollback                 # List available rollback versions
nvs rollback 0               # Rollback to most recent previous
```

### Configuration Switching

Manage multiple Neovim configurations:

```bash
nvs config                   # Interactive selection
nvs config nvim-test         # Direct switch
```

### Scripting Support

All listing and status commands support `--json` for machine-readable output:

```bash
nvs list --json              # Installed versions as JSON
nvs current --json           # Current version details as JSON
nvs doctor --json            # System checks as JSON
```

---

## System Requirements

| Platform | Architecture          | Status             |
| -------- | --------------------- | ------------------ |
| macOS    | Intel (amd64)         | ✅ Fully supported |
| macOS    | Apple Silicon (arm64) | ✅ Fully supported |
| Linux    | amd64                 | ✅ Fully supported |
| Linux    | arm64                 | ✅ Fully supported |
| Windows  | amd64                 | ✅ Fully supported |
| Windows  | arm64                 | ✅ Fully supported |

**Base dependencies:** `git`, `curl`, `tar`  
**Build dependencies:** `make`, `cmake`, `gettext`, `ninja` (nvs checks for these automatically)

---

## Documentation

| Document                                   | Description                                  |
| ------------------------------------------ | -------------------------------------------- |
| [Installation Guide](docs/INSTALLATION.md) | All installation methods and troubleshooting |
| [Usage Guide](docs/USAGE.md)               | Complete command reference with examples     |
| [Configuration](docs/CONFIGURATION.md)     | Environment setup and customization          |
| [Development](docs/DEVELOPMENT.md)         | Architecture and development setup           |
| [Contributing](docs/CONTRIBUTING.md)       | How to contribute to nvs                     |

---

## Contributing

Contributions are welcome! See the [Contributing Guide](docs/CONTRIBUTING.md) to get started.

```bash
git clone https://github.com/y3owk1n/nvs.git
cd nvs
just test    # Run tests
just lint    # Run linter
just build   # Build binary
```

---

## License

[MIT License](LICENSE) © Kyle Wong
