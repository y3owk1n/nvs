# Usage Guide

This guide provides comprehensive instructions for using **nvs** to manage Neovim installations.

## Command Overview

**nvs** uses a clean subcommand interface. Run `nvs --help` for full details.

```bash
A CLI tool to install, switch, list, uninstall, and reset Neovim versions.

Usage:
  nvs [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      Switch Neovim configuration
  current     Show current active version with details
  env         Print NVS env configurations
  help        Help about any command
  install     Install a Neovim version or commit
  list        List installed versions
  list-remote List available remote versions with installation status (cached for 5 minutes or force)
  path        Automatically add the global binary directory to your PATH
  reset       Reset all data (remove symlinks, downloaded versions, cache, etc.)
  uninstall   Uninstall a specific version
  upgrade     Upgrade installed stable and/or nightly versions
  use         Switch to a specific version or commit hash

Flags:
  -h, --help      help for nvs
  -v, --verbose   Enable verbose logging
       --version   version for nvs

Use "nvs [command] --help" for more information about a command.
```

## Core Commands

### Installing Neovim Versions

#### `nvs install`

Install a specific Neovim version or build from commits.

> [!warning]
> To build from commit, ensure your system has `git`, `make`, and `cmake` installed.

> [!warning]
> On macOS, also run: `brew install ninja cmake gettext curl`

```bash
# Install latest stable release
nvs install stable

# Install latest nightly release
nvs install nightly

# Install specific version
nvs install v0.10.3
nvs install 0.10.3  # v prefix optional

# Build and install from latest master commit
nvs install master

# Build and install specific commit (7 or 40 characters)
nvs install 2db1ae3
nvs install 2db1ae37f14d71d1391110fe18709329263c77c9

# Shorthand
nvs i stable
```

### Switching Versions

#### `nvs use`

Switch to a particular version. Updates the global symlink so `nvim` command uses the selected version.

> [!note]
> **nvs** will attempt to install the version if it's not already installed.

```bash
nvs use stable
nvs use nightly
nvs use v0.10.3
nvs use 0.10.3
nvs use 2db1ae3
```

> [!warning]
> On Windows, you may need administrator privileges for symlink creation.

### Listing Versions

#### `nvs list-remote`

List available remote releases with installation status. Results are cached for 5 minutes to avoid GitHub rate limits.

> [!note]
> Only shows nightly, stable, and versions above v0.5.0.

```bash
nvs list-remote          # Use cached results
nvs list-remote force    # Force refresh cache

# Shorthand
nvs ls-remote
nvs ls-remote force
```

#### `nvs list`

List all locally installed Neovim versions.

```bash
nvs list

# Shorthand
nvs ls
```

### Checking Current Version

#### `nvs current`

Display the currently active Neovim version with details.

```bash
nvs current
```

### Upgrading Versions

#### `nvs upgrade`

Upgrade installed stable and/or nightly versions. Checks if already up-to-date before upgrading.

> [!note]
> Compares stored identifiers (release tag for stable, commit hash/date for nightly).

```bash
nvs upgrade         # Upgrade both stable and nightly if installed
nvs upgrade stable  # Upgrade only stable
nvs upgrade nightly # Upgrade only nightly

# Shorthand
nvs up
nvs up stable
nvs up nightly
```

## Configuration Management

### `nvs config`

Switch between multiple Neovim configurations. Scans `~/.config` for directories containing "nvim".

> [!note]
> Examples: `nvim`, `nvim-test`, `nvim-vanilla`

```bash
nvs config          # Interactive selection
nvs config nvim-test # Direct switch

# Shorthand
nvs c
nvs conf
```

## Maintenance Commands

### `nvs uninstall`

Remove an installed version. Prompts for confirmation if uninstalling the current version.

```bash
nvs uninstall stable
nvs uninstall nightly
nvs uninstall v0.10.3
nvs uninstall 0.10.3
nvs uninstall 2db1ae3

# Shorthand
nvs rm stable
nvs remove nightly
nvs un 0.10.3
```

### `nvs reset`

Reset to factory state. Removes all data, symlinks, and binaries.

> [!warning]
> This deletes all configuration, cache, and binary data. Use with caution.

```bash
nvs reset
```

## Utility Commands

### `nvs env`

Print environment configuration variables.

```bash
nvs env
```

### `nvs path`

Automatically add the global binary directory to your PATH.

> [!note]
> May not work with Nix Home Manager. See [Configuration Guide](CONFIGURATION.md#nix-home-manager) for alternatives.

```bash
nvs path
```

## Shell Integration

### Completions

**nvs** supports shell completions for bash, zsh, fish, and PowerShell.

```bash
# Generate completion script
nvs completion [bash|zsh|fish|powershell]
```

#### Bash

Add to `~/.bashrc` or `~/.bash_profile`:

```bash
source <(nvs completion bash)
```

#### Zsh

Ensure completion is enabled in `~/.zshrc`:

```bash
autoload -U compinit && compinit
```

Then add:

```bash
source <(nvs completion zsh)
```

#### Fish

For temporary use:

```bash
nvs completion fish | source
```

For permanent:

```bash
nvs completion fish > ~/.config/fish/completions/nvs.fish
```

## Examples

### Basic Workflow

```bash
# Install stable
nvs install stable

# Switch to it
nvs use stable

# Check version
nvs current

# Install nightly for testing
nvs install nightly

# Switch between versions
nvs use stable
nvs use nightly
```

### Managing Multiple Configs

```bash
# Assuming you have ~/.config/nvim and ~/.config/nvim-test
nvs config          # Select interactively
nvs config nvim-test # Switch directly
```

### Upgrading

```bash
# Upgrade all
nvs upgrade

# Upgrade specific
nvs upgrade nightly
```

## Troubleshooting

### Verbose Logging

Add `--verbose` or `-v` to any command for detailed logs:

```bash
nvs install stable --verbose
nvs use nightly -v
```

### Common Issues

- **Permission denied:** Ensure write access to binary directory
- **Version not found:** Check `nvs list-remote` for available versions
- **Symlink errors:** On Windows, try running as administrator
- **PATH issues:** Verify `nvs path` worked or manually add to PATH

## Related Documentation

- [Installation Guide](INSTALLATION.md) - How to install nvs
- [Configuration Guide](CONFIGURATION.md) - Environment setup
- [Development Guide](DEVELOPMENT.md) - For contributors
- [Contributing Guide](CONTRIBUTING.md) - How to contribute
