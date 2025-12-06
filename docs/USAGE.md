# Usage Guide

Complete command reference for **nvs** – the Neovim Version Switcher.

---

## Quick Reference

| Command                   | Description                     |
| ------------------------- | ------------------------------- |
| `nvs install <version>`   | Install a version               |
| `nvs install --pick`      | Install with interactive picker |
| `nvs use <version>`       | Switch to a version             |
| `nvs use --pick`          | Switch with interactive picker  |
| `nvs list`                | List installed versions         |
| `nvs list-remote`         | List available versions         |
| `nvs current`             | Show active version             |
| `nvs upgrade [version]`   | Upgrade installed versions      |
| `nvs upgrade --pick`      | Upgrade with interactive picker |
| `nvs uninstall <version>` | Remove a version                |
| `nvs uninstall --pick`    | Remove with interactive picker  |
| `nvs pin [version]`       | Pin version to directory        |
| `nvs pin --pick`          | Pin with interactive picker     |
| `nvs rollback [index]`    | Rollback nightly version        |
| `nvs run <version>`       | Run version without switching   |
| `nvs run --pick`          | Run with interactive picker     |
| `nvs config [name]`       | Switch Neovim config            |
| `nvs doctor`              | System health check             |
| `nvs hook <shell>`        | Generate auto-switch hook       |
| `nvs env`                 | Print environment config        |

**Shorthands:** `i` (install), `ls` (list), `ls-remote` (list-remote), `rm`/`un` (uninstall), `up` (upgrade), `c`/`conf` (config)

---

## Table of Contents

- [Version Specifiers](#version-specifiers)
- [Installing Versions](#installing-versions)
- [Switching Versions](#switching-versions)
- [Listing Versions](#listing-versions)
- [Upgrading Versions](#upgrading-versions)
- [Version Pinning](#version-pinning)
- [Nightly Management](#nightly-management)
- [Configuration Switching](#configuration-switching)
- [Shell Integration](#shell-integration)
- [Utility Commands](#utility-commands)
- [Common Workflows](#common-workflows)

---

## Version Specifiers

**nvs** supports multiple version formats:

| Format     | Example               | Description                                                        |
| ---------- | --------------------- | ------------------------------------------------------------------ |
| `stable`   | `nvs install stable`  | Latest stable release                                              |
| `nightly`  | `nvs install nightly` | Latest nightly build                                               |
| `vX.Y.Z`   | `nvs install v0.10.3` | Specific version tag                                               |
| `X.Y.Z`    | `nvs install 0.10.3`  | Version without `v` prefix                                         |
| `master`   | `nvs install master`  | Build from latest master commit (resolves to specific commit hash) |
| `<commit>` | `nvs install 2db1ae3` | Build from specific commit (7+ chars)                              |

---

## Installing Versions

### `nvs install <version>`

Download and install a Neovim version.

```bash
# Install releases
nvs install stable
nvs install nightly
nvs install v0.10.3
nvs install 0.10.3        # v prefix optional

# Build from source
nvs install master        # Latest master commit (resolves to specific hash)
nvs install 2db1ae3       # Short commit hash
nvs install 2db1ae37f14d71d1391110fe18709329263c77c9  # Full hash

# Interactive selection
nvs install --pick        # Choose from available remote versions

# Shorthand
nvs i stable
```

> [!NOTE]
> **Base dependencies** (required): `git`, `curl`, `tar`
> **Build dependencies** (for source builds): `make`, `cmake`, `gettext`, `ninja`
> nvs automatically checks for these dependencies. Run `nvs doctor` for detailed status.
> Build operations show real-time progress with elapsed time and status updates.

**Flags:**

- `--pick`, `-p` – Launch interactive picker to select version from available remote releases
- `--verbose`, `-v` – Enable detailed logging

---

## Switching Versions

### `nvs use <version>`

Switch to an installed version. Creates/updates the global `nvim` symlink.

```bash
nvs use stable
nvs use nightly
nvs use v0.10.3
nvs use 2db1ae3

# Interactive selection
nvs use --pick

# Without arguments, reads from .nvs-version file
nvs use
```

> [!TIP]
> If the version isn't installed, **nvs** will attempt to install it automatically.

**Flags:**

- `--pick`, `-p` – Launch interactive picker to select version from installed versions
- `--force`, `-f` – Skip the running Neovim check

> [!WARNING]
> **nvs** warns if Neovim is running to prevent issues. Use `--force` to switch anyway.

---

### `nvs run <version> [-- args...]`

Run a specific version without changing the global symlink. Useful for testing.

```bash
nvs run stable
nvs run nightly -- --clean
nvs run v0.10.3 -- -c "checkhealth"
nvs run nightly -- file.txt

# Interactive selection
nvs run --pick -- --clean
```

> [!NOTE]
> The version must already be installed. Arguments after `--` are passed to Neovim.

**Flags:**

- `--pick`, `-p` – Launch interactive picker to select version from installed versions

---

## Listing Versions

### `nvs list`

Show all installed versions on your system.

```bash
nvs list
nvs ls      # Shorthand
nvs list --json  # JSON output
```

**Output example:**

```text
   VERSION    STATUS
------------------------
 → nightly  Current
 stable     Installed
```

**JSON output example:**

```json
{
  "versions": [
    {
      "name": "nightly",
      "status": "current",
      "type": "nightly"
    },
    {
      "name": "stable",
      "status": "installed",
      "type": "stable"
    }
  ]
}
```

---

### `nvs list-remote`

Show available versions from GitHub. Results are cached for 5 minutes.

```bash
nvs list-remote
nvs ls-remote           # Shorthand
nvs list-remote --force # Bypass cache
nvs list-remote --json  # JSON output
```

**Output example:**

```text
    TAG       STATUS                     DETAILS
------------------------------------------------------------------
  nightly  Current (↑)    Published: 2025-12-05, Commit: 903335a
  stable   Installed      stable version: v0.11.5
  v0.11.5  Not Installed
  v0.10.4  Not Installed
  ...
```

**JSON output example:**

```json
{
  "releases": [
    {
      "tag": "nightly",
      "status": "current",
      "details": "Published: 2025-12-05, Commit: 903335a",
      "prerelease": true
    },
    {
      "tag": "stable",
      "status": "installed",
      "details": "stable version: v0.11.5",
      "prerelease": false
    }
  ]
}
```

**Status meanings:**

- `Current` – Currently active version
- `Installed` – Version is installed locally
- `Current (↑)` or `Installed (↑)` – Upgrade available
- `Not Installed` – Available to install

---

### `nvs current`

Display the currently active Neovim version with details.

```bash
nvs current
nvs current --json  # JSON output
```

**Output example:**

```text
ℹ nightly
  Published: 2025-12-05
  Commit: 903335a
```

**JSON output example:**

```json
{
  "name": "nightly",
  "type": "nightly",
  "commit": "903335a",
  "published": "2025-12-05"
}
```

---

## Upgrading Versions

### `nvs upgrade [stable|nightly]`

Upgrade installed stable and/or nightly versions to the latest release.

```bash
nvs upgrade             # Upgrade both
nvs upgrade stable      # Upgrade stable only
nvs upgrade nightly     # Upgrade nightly only
nvs upgrade --pick      # Interactive selection
nvs up                  # Shorthand
```

> [!NOTE]
> Compares stored identifiers (release tag for stable, commit hash for nightly) to determine if an upgrade is needed.

When upgrading nightly, a changelog of commits since your last version is displayed.

**Flags:**

- `--pick`, `-p` – Launch interactive picker to select which versions to upgrade

---

## Removing Versions

### `nvs uninstall <version>`

Remove an installed Neovim version from your system.

```bash
nvs uninstall stable
nvs uninstall nightly
nvs uninstall v0.10.3
nvs uninstall --pick        # Interactive selection
nvs rm stable               # Shorthand
nvs un nightly              # Shorthand
```

> [!WARNING]
> If the version being uninstalled is currently active, you'll be prompted to confirm and optionally switch to another version.

**Flags:**

- `--pick`, `-p` – Launch interactive picker to select version from installed versions

---

## Version Pinning

### `nvs pin [version]`

Create a `.nvs-version` file to pin a version to the current directory.

```bash
nvs pin stable          # Pin stable
nvs pin nightly         # Pin nightly
nvs pin v0.10.3         # Pin specific version
nvs pin --pick          # Interactive selection
nvs pin                 # Pin current version
nvs pin -g stable       # Pin globally (~/.nvs-version)
```

**Flags:**

- `--pick`, `-p` – Launch interactive picker to select version from installed versions
- `--global`, `-g` – Create pin file in home directory

**How it works:**

1. Creates `.nvs-version` file with the version identifier
2. When `nvs use` is run without arguments, it reads from this file
3. With auto-switching enabled, version changes automatically on `cd`

---

## Nightly Management

### `nvs rollback [index]`

Rollback to a previously installed nightly version.

```bash
nvs rollback            # List available versions
nvs rollback 0          # Rollback to most recent previous
nvs rollback 2          # Rollback to specific index
```

**How it works:**

- Previous nightly versions are automatically saved during upgrades
- Up to 5 previous versions are kept by default
- Rollback replaces the current nightly with the selected version

---

## Configuration Switching

### `nvs config [name]`

Switch between multiple Neovim configurations. Scans `~/.config` for directories containing "nvim".

```bash
nvs config              # Interactive selection
nvs config nvim-test    # Direct switch
nvs c                   # Shorthand
nvs conf                # Shorthand
```

**Example configurations:**

- `~/.config/nvim` – Main configuration
- `~/.config/nvim-test` – Testing configuration
- `~/.config/nvim-minimal` – Minimal setup

> [!NOTE]
> This modifies the `NVIM_APPNAME` environment variable or symlinks configuration directories.

---

## Shell Integration

### `nvs hook <shell>`

Generate shell hook code for automatic version switching. When enabled, **nvs** automatically switches versions when entering a directory with a `.nvs-version` file.

**Bash** (`~/.bashrc`):

```bash
eval "$(nvs hook bash)"
```

**Zsh** (`~/.zshrc`):

```zsh
eval "$(nvs hook zsh)"
```

**Fish** (`~/.config/fish/config.fish`):

```fish
nvs hook fish | source
```

---

### `nvs env`

Print environment configuration. Useful for debugging or manual setup.

```bash
nvs env                     # Show current config
nvs env --json              # JSON output
nvs env --source            # Output for shell eval
nvs env --source --shell zsh  # Specify shell explicitly
```

**Output example:**

```text
   VARIABLE     VALUE
----------------------------------
 NVS_CONFIG_DIR  /home/user/.config/nvs
 NVS_CACHE_DIR   /home/user/.cache/nvs
 NVS_BIN_DIR     /home/user/.local/bin
```

**JSON output example:**

```json
{
  "NVS_CONFIG_DIR": "/home/user/.config/nvs",
  "NVS_CACHE_DIR": "/home/user/.cache/nvs",
  "NVS_BIN_DIR": "/home/user/.local/bin"
}
```

---

### `nvs completion <shell>`

Generate shell completion scripts.

```bash
nvs completion bash
nvs completion zsh
nvs completion fish
nvs completion powershell
```

**Setup:**

**Bash** (`~/.bashrc`):

```bash
source <(nvs completion bash)
```

**Zsh** (`~/.zshrc`):

```zsh
autoload -U compinit && compinit
source <(nvs completion zsh)
```

**Fish:**

```fish
nvs completion fish > ~/.config/fish/completions/nvs.fish
```

---

## Utility Commands

### `nvs doctor`

Check system health and diagnose potential issues.

```bash
nvs doctor
nvs doctor --json  # JSON output
```

**Output example:**

```text
Checking Shell... ✓
Checking Environment variables... ✓
Checking PATH... ✓
Checking Dependencies... ✓
Checking Permissions... ✓
No issues found! You are ready to go.
```

**JSON output example:**

```json
{
  "checks": [
    {
      "name": "Shell",
      "status": "ok"
    },
    {
      "name": "Environment variables",
      "status": "ok"
    }
  ],
  "issues": []
}
```

**Checks performed:**

- OS/Architecture compatibility
- Shell detection
- Environment variables (`NVS_CONFIG_DIR`, `NVS_CACHE_DIR`, `NVS_BIN_DIR`, `PATH`)
- Required dependencies (`git`, `curl`, `tar`)
- Directory permissions

---

### `nvs path`

Automatically add the binary directory to your shell's `PATH`.

```bash
nvs path
```

> [!NOTE]
> May not work with Nix Home Manager. See [Configuration Guide](CONFIGURATION.md#nix-home-manager) for alternatives.

---

### `nvs reset`

Reset to factory state. Removes all configuration, cache, installed versions, and symlinks.

```bash
nvs reset
```

> [!CAUTION]
> This is destructive and cannot be undone. You will need to reinstall all Neovim versions.

---

## Common Workflows

### Daily Development

```bash
# Start with stable
nvs use stable
nvim .

# Test feature in nightly
nvs run nightly -- --clean

# Switch back
nvs use stable
```

### Project-Specific Versions

```bash
# In project directory
cd my-project
nvs pin v0.9.5

# Auto-loads on cd (with hook enabled)
cd ../other-project
cd my-project  # Now using v0.9.5
```

### Keeping Up to Date

```bash
# Check what's available
nvs list-remote

# Upgrade everything
nvs upgrade

# If nightly breaks, rollback
nvs rollback 0
```

### Testing Neovim Changes

```bash
# Build from specific commit
nvs install abc1234
nvs use abc1234

# Test
nvim -c "lua print(vim.version())"

# Return to stable
nvs use stable
```

### Multiple Configurations

```bash
# Work setup
nvs config nvim-work

# Personal setup
nvs config nvim

# Minimal testing
nvs config nvim-minimal
```

---

## Global Flags

These flags work with any command:

| Flag              | Description                   |
| ----------------- | ----------------------------- |
| `--verbose`, `-v` | Enable detailed debug logging |
| `--help`, `-h`    | Show help for command         |
| `--version`       | Show nvs version              |

---

## Next Steps

- [Environment Configuration →](CONFIGURATION.md)
- [Installation Methods →](INSTALLATION.md)
- [Contributing →](CONTRIBUTING.md)
