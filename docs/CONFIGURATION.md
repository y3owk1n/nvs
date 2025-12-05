# Configuration Guide

This guide explains how to configure **nvs** for optimal use, including environment setup, directory customization, and PATH configuration.

## Environment Setup

Before using **nvs**, you must set up environment variables. This is done by evaluating the output of `nvs env --source`.

### Automatic Setup

The `--source` flag detects your shell and prints the correct syntax:

```bash
eval "$(nvs env --source)"
```

If auto-detection fails, specify your shell explicitly:

```bash
nvs env --source --shell bash
nvs env --source --shell zsh
nvs env --source --shell fish
```

### Manual Shell Configuration

#### Bash

Add to `~/.bashrc` or `~/.bash_profile`:

```bash
eval "$(nvs env --source)"
```

#### Zsh

Add to `~/.zshrc`:

```zsh
eval "$(nvs env --source)"
```

#### Fish

Create `~/.config/fish/conf.d/nvs.fish`:

```fish
nvs env --source | source
```

> [!note]
> Explicit shell specification avoids detection issues across different environments.

## Directory Structure

**nvs** follows OS-specific conventions for storing data:

### Default Locations

- **Configuration:** `~/.config/nvs` (Unix) / `AppData\Roaming\nvs` (Windows)
- **Cache:** `~/.cache/nvs` (Unix) / `AppData\Local\nvs` (Windows)
- **Binaries:** `~/.local/bin` (Unix) / `AppData\Local\Programs` (Windows)

> [!note]
> Directories are created automatically on first command execution.

### Environment Variables

Override defaults with these variables:

| Variable            | Description          | Default (Unix)  | Default (Windows)        |
| ------------------- | -------------------- | --------------- | ------------------------ |
| `NVS_CONFIG_DIR`    | nvs configuration    | `~/.config/nvs` | `AppData\Roaming\nvs`    |
| `NVS_CACHE_DIR`     | Cache files          | `~/.cache/nvs`  | `AppData\Local\nvs`      |
| `NVS_BIN_DIR`       | Binary symlinks      | `~/.local/bin`  | `AppData\Local\Programs` |
| `NVS_GITHUB_MIRROR` | GitHub mirror URL    | (none)          | (none)                   |

### Setting Custom Directories

#### Unix-like Systems

Add to `~/.bashrc`, `~/.zshrc`, etc.:

```bash
export NVS_CONFIG_DIR="$HOME/custom-config/nvs"
export NVS_CACHE_DIR="$HOME/custom-cache/nvs"
export NVS_BIN_DIR="$HOME/custom-bin"
```

Reload configuration:

```bash
source ~/.bashrc  # or ~/.zshrc
```

#### Windows

Temporary (current session):

```cmd
set NVS_CONFIG_DIR=C:\Path\To\Custom\Config
set NVS_CACHE_DIR=C:\Path\To\Custom\Cache
set NVS_BIN_DIR=C:\Path\To\Custom\Bin
```

Permanent (via Command Prompt as admin):

```cmd
setx NVS_CONFIG_DIR "C:\Path\To\Custom\Config"
setx NVS_CACHE_DIR "C:\Path\To\Custom\Cache"
setx NVS_BIN_DIR "C:\Path\To\Custom\Bin"
```

Or use System Properties → Advanced → Environment Variables.

> [!note]
> When overriding `NVS_BIN_DIR`, update your PATH accordingly.

## GitHub Mirror

If you have limited access to GitHub (e.g., in certain regions), you can use a GitHub mirror for downloading Neovim releases:

```bash
export NVS_GITHUB_MIRROR="https://mirror.ghproxy.com"
```

This replaces `https://github.com` in download URLs while still using the official GitHub API for release information.

> [!note]
> The mirror only affects download URLs, not API calls. Common mirrors include `https://mirror.ghproxy.com` and `https://ghproxy.net`.

## PATH Configuration

To use Neovim binaries installed by **nvs**, add the binary directory to your PATH.

### Automatic PATH Setup

**nvs** provides a `path` command for automatic setup:

```bash
nvs path
```

> [!note]
> This may not work with Nix Home Manager. See below for alternatives.

### Manual PATH Setup

#### macOS/Linux

##### Bash

Add to `~/.bashrc` or `~/.bash_profile`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

##### Zsh

Add to `~/.zshrc`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

##### Fish

Add to `~/.config/fish/config.fish`:

```fish
set -gx PATH $HOME/.local/bin $PATH
```

#### Windows

Add to PATH environment variable. Example for current session:

```cmd
set PATH=%USERPROFILE%\AppData\Local\Programs\nvim\bin;%PATH%
```

> [!note]
> Target the nested `bin` folder as Windows requires the full path structure.

### Nix Home Manager

If using Nix Home Manager, add to your configuration:

```nix
{
  home.sessionPath = [
    "$HOME/.local/bin"
  ];
}
```

Reference: [Nix Home Manager Documentation](https://nix-community.github.io/home-manager/options.xhtml#opt-home.sessionPath)

## Configuration Files

**nvs** stores configuration in the config directory:

- Version information and preferences
- Installation records
- User settings

The config directory is separate from Neovim's config directory (`~/.config/nvim`).

## Cache Management

Cache directory stores:

- Downloaded release information (5-minute TTL)
- Temporary build artifacts
- Metadata for installed versions

Clear cache manually by deleting the cache directory or using `nvs reset`.

## Troubleshooting

### Environment Issues

- **Command not found:** Verify PATH includes binary directory
- **Permission denied:** Check write permissions for config/cache/bin directories
- **Shell detection fails:** Use `--shell` flag explicitly

### Directory Issues

- **Custom paths not working:** Ensure environment variables are exported before running commands
- **Old directories persist:** Manually delete default directories after setting custom paths

### PATH Issues

- **nvs path fails:** Manually add to PATH as shown above
- **Binary not found:** Verify `NVS_BIN_DIR` is in PATH and symlinks exist

## Advanced Configuration

### Multiple Environments

For different environments (work/personal), use different config directories:

```bash
export NVS_CONFIG_DIR="$HOME/.config/nvs-work"
export NVS_CACHE_DIR="$HOME/.cache/nvs-work"
export NVS_BIN_DIR="$HOME/.local/bin-work"
```

### CI/CD Usage

For automated environments, set minimal configuration:

```bash
export NVS_CONFIG_DIR="/tmp/nvs-config"
export NVS_CACHE_DIR="/tmp/nvs-cache"
export NVS_BIN_DIR="$HOME/bin"
```

## Related Documentation

- [Installation Guide](INSTALLATION.md) - How to install nvs
- [Usage Guide](USAGE.md) - Command reference
- [Development Guide](DEVELOPMENT.md) - For contributors
- [Contributing Guide](CONTRIBUTING.md) - How to contribute

