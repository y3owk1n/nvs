# nvs (Neovim Version Switcher)

**Neovim Version Switcher** – Easily install, switch, and manage multiple versions (including commit hashes) and config of Neovim like a boss 🚀

[![CI](https://github.com/y3owk1n/nvs/actions/workflows/ci.yml/badge.svg)](https://github.com/y3owk1n/nvs/actions/workflows/ci.yml) [![GitHub release](https://img.shields.io/github/release/y3owk1n/nvs.svg)](https://github.com/y3owk1n/nvs/releases) [![License](https://img.shields.io/github/license/y3owk1n/nvs.svg)](LICENSE)

## Overview

**nvs** (Neovim Version Switcher/Manager) is a lightweight cross-platform CLI tool written in Go that makes it super easy to install, switch between, and manage multiple versions of Neovim and config on your machine. Whether you're testing a cutting-edge nightly build or sticking with the stable release, **nvs** has got your back!

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

## Quick Start

1. **Install nvs:**
   ```bash
   curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/install.sh | bash
   ```

2. **Set up environment:**
   ```bash
   eval "$(nvs env --source)"
   ```

3. **Install and use Neovim versions:**
   ```bash
   nvs install stable
   nvs use stable
   ```

## Documentation

- [Installation Guide](docs/INSTALLATION.md) - Detailed installation instructions
- [Usage Guide](docs/USAGE.md) - Complete command reference and examples
- [Configuration](docs/CONFIGURATION.md) - Environment setup and customization
- [Development](docs/DEVELOPMENT.md) - Contributing and development setup
- [Contributing](docs/CONTRIBUTING.md) - How to contribute to the project

## Features

- **Easy Installation:** Download and install Neovim versions directly from GitHub
- **Version Switching:** Switch between installed versions instantly
- **Config Switching:** Toggle between Neovim configurations
- **Cross-Platform:** Works on macOS, Linux, and Windows
- **Verbose Logging:** Detailed logs with `--verbose` flag

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
