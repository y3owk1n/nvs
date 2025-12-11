# Configuration Guide

Environment setup and customization options for **nvs**.

---

## Quick Reference

| Variable            | Description         | Default (Unix)  |
| ------------------- | ------------------- | --------------- |
| `NVS_CONFIG_DIR`    | Configuration files | `~/.config/nvs` |
| `NVS_CACHE_DIR`     | Cache files         | `~/.cache/nvs`  |
| `NVS_BIN_DIR`       | Binary symlinks     | `~/.local/bin`  |
| `NVS_GITHUB_MIRROR` | GitHub mirror URL   | (none)          |
| `NVS_USE_GLOBAL_CACHE` | Use global cache for releases | `false` |

---

## Table of Contents

- [Shell Setup](#shell-setup)
- [Environment Variables](#environment-variables)
- [Directory Structure](#directory-structure)
- [GitHub Mirror](#github-mirror)
- [PATH Configuration](#path-configuration)
- [Nix / Home Manager](#nix--home-manager)
- [Advanced Configuration](#advanced-configuration)

---

## Shell Setup

**nvs** requires shell integration to function properly. Add the following to your shell configuration:

### Bash

Add to `~/.bashrc` or `~/.bash_profile`:

```bash
# nvs environment setup
eval "$(nvs env --source)"

# Optional: Shell completions
source <(nvs completion bash)

# Optional: Auto-switch on cd
eval "$(nvs hook bash)"
```

### Zsh

Add to `~/.zshrc`:

```zsh
# nvs environment setup
eval "$(nvs env --source)"

# Optional: Shell completions
autoload -U compinit && compinit
source <(nvs completion zsh)

# Optional: Auto-switch on cd
eval "$(nvs hook zsh)"
```

### Fish

Add to `~/.config/fish/config.fish`:

```fish
# nvs environment setup
nvs env --source | source

# Optional: Auto-switch on cd
nvs hook fish | source
```

For completions, run once:

```fish
nvs completion fish > ~/.config/fish/completions/nvs.fish
```

### PowerShell

Add to your `$PROFILE`:

```powershell
# nvs environment setup
nvs env --source | Invoke-Expression

# Optional: Shell completions
nvs completion powershell | Out-String | Invoke-Expression
```

### Manual Shell Specification

If auto-detection fails, specify your shell explicitly:

```bash
nvs env --source --shell bash
nvs env --source --shell zsh
nvs env --source --shell fish
nvs env --source --shell powershell
```

---

## Environment Variables

### NVS_CONFIG_DIR

**Purpose:** Configuration storage (version records, settings)

**Default:**

- Unix: `~/.config/nvs`
- Windows: `%APPDATA%\nvs`

**Example:**

```bash
export NVS_CONFIG_DIR="$HOME/.nvs/config"
```

---

### NVS_CACHE_DIR

**Purpose:** Cache storage (release info, build artifacts)

**Default:**

- Unix: `~/.cache/nvs`
- Windows: `%LOCALAPPDATA%\nvs`

**Example:**

```bash
export NVS_CACHE_DIR="$HOME/.nvs/cache"
```

> [!TIP]
> Cache is automatically cleared when stale. Release info is cached for 5 minutes.

---

### NVS_BIN_DIR

**Purpose:** Location for the `nvim` symlink

**Default:**

- Unix: `~/.local/bin`
- Windows: `%LOCALAPPDATA%\Programs`

**Example:**

```bash
export NVS_BIN_DIR="$HOME/bin"
```

> [!IMPORTANT]
> Ensure `NVS_BIN_DIR` is in your `PATH` for the `nvim` command to work.

---

### NVS_GITHUB_MIRROR

**Purpose:** Use a GitHub mirror for downloading releases (useful in restricted regions)

**Default:** None (uses `github.com` directly)

**Example:**

```bash
export NVS_GITHUB_MIRROR="https://mirror.ghproxy.com"
```

**Common mirrors:**

- `https://mirror.ghproxy.com`
- `https://ghproxy.net`

> [!NOTE]
> The mirror only affects download URLs. API calls still go to GitHub directly.

---

### NVS_USE_GLOBAL_CACHE

**Purpose:** Enable fetching Neovim releases from a global cache to reduce API calls and improve performance.

**Default:** `false`

**Example:**

```bash
export NVS_USE_GLOBAL_CACHE=true
```

**How it works:**

- When enabled, `nvs ls-remote` fetches releases from a pre-built JSON cache hosted in the nvs repository.
- This cache is updated daily via GitHub Actions, reducing GitHub API rate limits.
- `--force` still bypasses the global cache and fetches directly from the API.
- When disabled, behavior is unchanged (uses local cache and API).

> [!TIP]
> Enable this for faster `ls-remote` and to help avoid rate limits in high-usage scenarios.

---

## Directory Structure

**nvs** creates the following directory structure:

```text
~/.config/nvs/           # NVS_CONFIG_DIR
└── versions/            # Installed Neovim versions
    ├── stable/
    ├── nightly/
    ├── v0.10.3/
    └── ...

~/.cache/nvs/            # NVS_CACHE_DIR
└── releases.json        # Cached release information

~/.local/bin/            # NVS_BIN_DIR
└── nvim -> versions/stable/bin/nvim  # Symlink to active version
```

All directories are created automatically on first run.

---

## GitHub Mirror

If you have limited access to GitHub (e.g., in certain regions), configure a mirror:

```bash
# Add to shell config
export NVS_GITHUB_MIRROR="https://mirror.ghproxy.com"
```

**How it works:**

- Download URLs: `https://github.com/...` → `https://mirror.ghproxy.com/https://github.com/...`
- API calls: Still use `api.github.com` for release information

---

## PATH Configuration

The `nvim` command must be accessible in your `PATH`.

### Automatic Setup

```bash
nvs path  # Adds NVS_BIN_DIR to PATH in shell config
```

### Manual Setup

**Bash/Zsh:**

```bash
export PATH="$HOME/.local/bin:$PATH"
```

**Fish:**

```fish
fish_add_path $HOME/.local/bin
```

**Windows (PowerShell as Admin):**

```powershell
[Environment]::SetEnvironmentVariable("Path", "$env:LOCALAPPDATA\Programs;$env:Path", "User")
```

---

## Nix / Home Manager

### Home Manager Module (Recommended)

Full integration with automatic environment setup:

```nix
{
  inputs.nvs.url = "github:y3owk1n/nvs";

  outputs = { home-manager, nvs, ... }: {
    homeConfigurations.your-username = home-manager.lib.homeManagerConfiguration {
      modules = [
        { nixpkgs.overlays = [ nvs.overlays.default ]; }
        nvs.homeManagerModules.default
        {
           programs.nvs = {
             enable = true;

             # All options with defaults:
             enableAutoSwitch = true;      # Auto-switch on cd
             enableShellIntegration = true; # Run nvs env --source
              useGlobalCache = false;       # Set to true to use global cache and reduce API calls

             # Custom directories (optional)
             configDir = "${config.xdg.configHome}/nvs";
             cacheDir = "${config.xdg.cacheHome}/nvs";
             binDir = "${config.home.homeDirectory}/.local/bin";

             # Shell-specific integration
             shellIntegration = {
               bash = true;
               zsh = true;
               fish = true;
             };
           };
        }
      ];
    };
  };
}
```

**The module handles:**

- Installing nvs
- Setting `NVS_CONFIG_DIR`, `NVS_CACHE_DIR`, `NVS_BIN_DIR`, and `NVS_USE_GLOBAL_CACHE` (only when `useGlobalCache = true`)
- Adding `binDir` to `home.sessionPath`
- Shell integration (`nvs env --source`)
- Auto-switch hooks (`nvs hook`)
- Directory creation

### Manual Nix Setup

If not using the module, set environment manually:

```nix
{
  home.sessionVariables = {
    NVS_CONFIG_DIR = "${config.xdg.configHome}/nvs";
    NVS_CACHE_DIR = "${config.xdg.cacheHome}/nvs";
    NVS_BIN_DIR = "${config.home.homeDirectory}/.local/bin";
  };

  home.sessionPath = [
    "${config.home.homeDirectory}/.local/bin"
  ];
}
```

---

## Advanced Configuration

### Multiple Environments

Use separate directories for work/personal setups:

```bash
# Work environment
export NVS_CONFIG_DIR="$HOME/.config/nvs-work"
export NVS_CACHE_DIR="$HOME/.cache/nvs-work"
export NVS_BIN_DIR="$HOME/.local/bin-work"
```

Create shell aliases:

```bash
alias nvs-work='NVS_CONFIG_DIR=~/.config/nvs-work NVS_BIN_DIR=~/.local/bin-work nvs'
```

### CI/CD Configuration

Minimal setup for automated environments:

```bash
export NVS_CONFIG_DIR="/tmp/nvs-config"
export NVS_CACHE_DIR="/tmp/nvs-cache"
export NVS_BIN_DIR="$HOME/bin"

nvs install stable
nvs use stable
```

### Docker

```dockerfile
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y curl git

# Install nvs
RUN curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/install.sh | bash

# Add to PATH
ENV PATH="/root/.local/bin:$PATH"

# Install Neovim
RUN nvs install stable && nvs use stable
```

---

## Troubleshooting

### Environment Not Loading

**Symptom:** `nvim: command not found` after setup

**Solutions:**

1. Restart your terminal
2. Source config manually: `source ~/.bashrc` (or equivalent)
3. Verify PATH: `echo $PATH | tr ':' '\n' | grep local`
4. Check nvs directory exists: `ls -la ~/.local/bin/nvim`

### Shell Detection Fails

**Symptom:** `nvs env --source` outputs wrong format

**Solution:** Specify shell explicitly:

```bash
nvs env --source --shell zsh
```

### Permission Denied

**Symptom:** Cannot create symlinks or directories

**Solutions:**

```bash
# Fix directory permissions
chmod 755 ~/.local/bin
mkdir -p ~/.config/nvs ~/.cache/nvs

# Or use different directories
export NVS_BIN_DIR="$HOME/bin"
```

### Conflicts with Other Version Managers

If you use other Neovim or version managers:

1. Remove other managers' PATH entries
2. Ensure `NVS_BIN_DIR` appears first in `PATH`
3. Use `nvs doctor` to check for conflicts

---

## Next Steps

- [Command Reference →](USAGE.md)
- [Installation Options →](INSTALLATION.md)
- [Contributing →](CONTRIBUTING.md)
