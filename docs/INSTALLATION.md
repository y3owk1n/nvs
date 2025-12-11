# Installation Guide

This guide covers all methods to install **nvs** on your system.

---

## TL;DR

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/install.sh | bash
eval "$(nvs env --source)"
nvs install stable && nvs use stable
```

**Windows (PowerShell):**

```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/y3owk1n/nvs/main/install.ps1" -OutFile "install.ps1"; .\install.ps1
nvs install stable; nvs use stable
```

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation Methods](#installation-methods)
  - [Install Script (Recommended)](#method-1-install-script-recommended)
  - [Homebrew](#method-2-homebrew-macoslinux)
  - [Pre-built Binaries](#method-3-pre-built-binaries)
  - [Nix Flakes](#method-4-nix-flakes)
  - [Build from Source](#method-5-build-from-source)
  - [GitHub Actions](#method-6-github-actions)
- [Post-Installation Setup](#post-installation-setup)
- [Upgrading nvs](#upgrading-nvs)
- [Uninstalling](#uninstalling)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### System Requirements

| Platform | Architecture          | Status             |
| -------- | --------------------- | ------------------ |
| macOS    | Intel (amd64)         | ✅ Fully supported |
| macOS    | Apple Silicon (arm64) | ✅ Fully supported |
| Linux    | amd64                 | ✅ Fully supported |
| Linux    | arm64                 | ✅ Fully supported |
| Windows  | amd64                 | ✅ Fully supported |
| Windows  | arm64                 | ✅ Fully supported |

### Permissions

- **Unix:** Write access to `~/.local/bin` (or custom `NVS_BIN_DIR`)
- **Windows:** Ability to create symlinks (may require administrator privileges)

> [!WARNING]
> For the best experience, remove any existing Neovim installations from your system and let **nvs** manage them instead.

---

## Installation Methods

### Method 1: Install Script (Recommended)

The install script automatically detects your OS and architecture.

#### macOS / Linux / WSL

```bash
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/install.sh | bash
```

#### Windows

> [!WARNING]
> Windows support is experimental. Please report issues and contribute improvements.

```powershell
# Download and run the installer
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/y3owk1n/nvs/main/install.ps1" -OutFile "install.ps1"
.\install.ps1
```

If you encounter an execution policy error:

```powershell
Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned
```

> [!TIP]
> Always review remote scripts before execution. View the [install.sh](https://github.com/y3owk1n/nvs/blob/main/install.sh) or [install.ps1](https://github.com/y3owk1n/nvs/blob/main/install.ps1) source.

---

### Method 2: Homebrew (macOS/Linux)

```bash
brew tap y3owk1n/tap
brew install y3owk1n/tap/nvs
```

---

### Method 3: Pre-built Binaries

1. Download the appropriate binary from [GitHub Releases](https://github.com/y3owk1n/nvs/releases)
2. Extract and move to a directory in your `PATH`
3. Make executable (Unix): `chmod +x nvs`

**Available binaries:**

- `nvs-darwin-amd64` – macOS Intel
- `nvs-darwin-arm64` – macOS Apple Silicon
- `nvs-linux-amd64` – Linux x86_64
- `nvs-linux-arm64` – Linux ARM64
- `nvs-windows-amd64.exe` – Windows x86_64
- `nvs-windows-arm64.exe` – Windows ARM64

---

### Method 4: Nix (Flakes)

**nvs** provides first-class Nix support with automatic shell completions.

#### Quick Start with Home Manager

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nvs.url = "github:y3owk1n/nvs"; # or "https://flakehub.com/f/y3owk1n/nvs/0.1"
  };

  outputs = { nixpkgs, home-manager, nvs, ... }: {
    homeConfigurations.your-username = home-manager.lib.homeManagerConfiguration {
      pkgs = nixpkgs.legacyPackages.aarch64-darwin;
      modules = [
        { nixpkgs.overlays = [ nvs.overlays.default ]; }
        nvs.homeManagerModules.default
        {
          programs.nvs = {
            enable = true;
            # Optional settings:
            # enableAutoSwitch = true;
            # enableShellIntegration = true;
            # configDir = "${config.xdg.configHome}/nvs";
            # cacheDir = "${config.xdg.cacheHome}/nvs";
            # binDir = "${config.home.homeDirectory}/.local/bin";
          };
        }
      ];
    };
  };
}
```

The Home Manager module automatically:

- Installs nvs and adds `binDir` to your `PATH`
- Sets up environment variables (`NVS_CONFIG_DIR`, `NVS_CACHE_DIR`, `NVS_BIN_DIR`)
- Enables shell integration and auto-switching
- Creates necessary directories

#### NixOS / nix-darwin (System-wide)

```nix
{
  inputs.nvs.url = "github:y3owk1n/nvs";

  outputs = { nixpkgs, nvs, ... }: {
    nixosConfigurations.your-host = nixpkgs.lib.nixosSystem {
      modules = [{
        nixpkgs.overlays = [ nvs.overlays.default ];
        environment.systemPackages = [
          pkgs.nvs        # Pre-built binary
          # pkgs.nvs-source # Or build from source
        ];
      }];
    };
  };
}
```

#### Direct Run (Testing)

```bash
nix run github:y3owk1n/nvs
nix run github:y3owk1n/nvs#source  # Build from source
```

---

### Method 5: Build from Source

Requires [Go](https://golang.org/dl/) v1.21 or later.

```bash
git clone https://github.com/y3owk1n/nvs.git
cd nvs
go build -o nvs ./main.go
```

Move the binary to a directory in your `PATH`:

```bash
mv nvs ~/.local/bin/
```

**Cross-compilation examples:**

```bash
# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o nvs-darwin-arm64 ./main.go

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o nvs-darwin-amd64 ./main.go

# Linux
GOOS=linux GOARCH=amd64 go build -o nvs-linux-amd64 ./main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o nvs-windows-amd64.exe ./main.go
```

---

### Method 6: GitHub Actions

Use the **nvs** action in your GitHub workflows to install nvs and Neovim versions automatically.

#### Basic Usage

```yaml
- uses: y3owk1n/nvs@main
  with:
    version: stable  # Install and use stable Neovim
```

#### Advanced Usage

```yaml
- uses: y3owk1n/nvs@main
  with:
    version: nightly  # Install nightly build
    install-nvs: true  # Install nvs (default: true)
    use-global-cache: true  # Use global cache to reduce API calls
```

#### Inputs

| Input            | Description                                      | Default | Required |
|------------------|--------------------------------------------------|---------|----------|
| `version`        | Neovim version to install/use (e.g., `stable`, `nightly`, `v0.10.3`) | `stable` | No |
| `install-nvs`    | Whether to install nvs                           | `true`  | No |
| `use-global-cache` | Use global cache for releases to reduce API calls | `false` | No |

#### Supported Platforms

- **Linux:** `ubuntu-latest`, `ubuntu-20.04`, etc.
- **macOS:** `macos-latest`, `macos-12`, etc.
- **Windows:** `windows-latest`, `windows-2022`, etc.

#### Example Workflow

```yaml
name: CI
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Neovim
        uses: y3owk1n/nvs@main
        with:
          version: stable
          use-global-cache: true

      - name: Verify Neovim
        run: nvim --version
```

The action automatically:
- Downloads and installs nvs using the official install scripts
- Installs the specified Neovim version
- Adds nvs and Neovim to the `PATH` for subsequent steps

> [!TIP]
> View the [action.yml](https://github.com/y3owk1n/nvs/blob/main/action.yml) source for implementation details.

---

## Post-Installation Setup

After installing **nvs**, configure your shell environment.

### Shell Configuration

Add the following to your shell configuration file:

**Bash** (`~/.bashrc` or `~/.bash_profile`):

```bash
eval "$(nvs env --source)"
```

**Zsh** (`~/.zshrc`):

```zsh
eval "$(nvs env --source)"
```

**Fish** (`~/.config/fish/config.fish`):

```fish
nvs env --source | source
```

**PowerShell** (`$PROFILE`):

```powershell
nvs env --source | Invoke-Expression
```

### Verify Installation

```bash
# Check nvs is working
nvs --version
nvs doctor

# Install your first version
nvs install stable
nvs use stable
nvim --version
```

### Enable Auto-Switching (Optional)

Automatically switch Neovim versions based on `.nvs-version` files:

**Bash/Zsh:**

```bash
eval "$(nvs hook bash)"  # or zsh
```

**Fish:**

```fish
nvs hook fish | source
```

---

## Upgrading nvs

**Install script:** Re-run the same installation command.

**Homebrew:**

```bash
brew upgrade y3owk1n/tap/nvs
```

**Nix:** Update your flake inputs and rebuild.

**Manual:** Download the new binary and replace the old one.

---

## Uninstalling

### Using Uninstall Scripts

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/uninstall.sh | bash
```

**Windows:**

```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/y3owk1n/nvs/main/uninstall.ps1" -OutFile "uninstall.ps1"
.\uninstall.ps1
```

### Manual Uninstall

1. Remove the nvs binary from your `PATH`
2. Delete configuration: `rm -rf ~/.config/nvs`
3. Delete cache: `rm -rf ~/.cache/nvs`
4. Delete installed versions: `rm -rf ~/.local/bin/nvim` (or your `NVS_BIN_DIR`)
5. Remove shell configuration lines added during setup

---

## Troubleshooting

### Common Issues

#### "Command not found: nvs"

Ensure the binary location is in your `PATH`:

```bash
echo $PATH | tr ':' '\n' | grep -E "(local/bin|nvs)"
```

Restart your terminal or source your shell config after installation.

#### Permission Denied

**Unix:** Ensure write access to the binary directory:

```bash
mkdir -p ~/.local/bin
chmod 755 ~/.local/bin
```

**Windows:** Run terminal as administrator for symlink creation.

#### Shell Not Detected

Specify your shell explicitly:

```bash
nvs env --source --shell bash
nvs env --source --shell zsh
nvs env --source --shell fish
```

#### Build Dependencies Missing

When building Neovim from source (commits), nvs automatically checks for the following required tools:

**Base dependencies** (required for all nvs operations): `git`, `curl`, `tar`
**Build dependencies** (required only for building from source): `make`, `cmake`, `gettext`, `ninja`

- Missing base dependencies will prevent nvs from working and show error messages
- Missing build dependencies will show warnings but allow nvs to work with pre-built releases
- Run `nvs doctor` to check your system's dependency status

**macOS:**

```bash
brew install ninja cmake gettext curl
```

**Ubuntu/Debian:**

```bash
sudo apt install ninja-build cmake gettext curl unzip
```

**Devbox (recommended for development):**

The devbox environment automatically provides all required dependencies. Run `devbox install` to set up the development environment.

#### Windows-Specific Issues

- Ensure PowerShell has appropriate permissions
- Check antivirus isn't blocking binary downloads
- Restart terminal after installation for `PATH` changes
- The installer automatically adds nvs to user `PATH`

### Getting Help

1. Run `nvs doctor` to diagnose issues
2. Use `nvs --verbose <command>` for detailed logs
3. Check [GitHub Issues](https://github.com/y3owk1n/nvs/issues)
4. Open a new issue with system info and verbose output

---

## Next Steps

- [Configure your environment →](CONFIGURATION.md)
- [Learn the commands →](USAGE.md)
- [Set up for development →](DEVELOPMENT.md)
