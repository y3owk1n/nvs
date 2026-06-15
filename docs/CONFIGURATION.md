# Configuration Guide

Environment setup and customization options for **nvs**.

---

## Quick Reference

| Variable               | Description                                       | Default (Unix)     |
| ---------------------- | ------------------------------------------------- | ------------------ |
| `NVS_CONFIG_DIR`       | Configuration files                               | `~/.config/nvs`    |
| `NVS_CACHE_DIR`        | Cache files                                       | `~/.cache/nvs`     |
| `NVS_BIN_DIR`          | Binary symlinks                                   | `~/.local/bin`     |
| `NVS_GITHUB_MIRROR`    | GitHub mirror URL                                 | (none)             |
| `NVS_USE_GLOBAL_CACHE` | Use global cache for releases                     | `false`            |
| `NVS_LOG`              | Developer log level (debug/info/warn/...)         | `warn`             |
| `NVS_LOG_FILE`         | Tee developer logs to a file                      | (none)             |
| `NVS_COLOR_*`          | Theme any palette color (see [Theming](#theming)) | (built-in palette) |
| `NO_COLOR`             | Disable all ANSI color output                     | (unset)            |
| `FORCE_COLOR`          | Force ANSI color even on non-TTY                  | (unset)            |

---

## Table of Contents

- [Shell Setup](#shell-setup)
- [Environment Variables](#environment-variables)
- [Theming](#theming)
- [Directory Structure](#directory-structure)
- [GitHub Mirror](#github-mirror)
- [PATH Configuration](#path-configuration)
- [Logging](#logging)
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

All `NVS_*` environment variables are validated at startup. An invalid value (e.g. a typo, a path with a control character, an unparseable log level) is reported once on stderr and the corresponding default is used — the program does not refuse to run, but the user is told why their setting had no effect.

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

**Recognized values** (case-insensitive):

| Truthy                   | Falsy                     |
| ------------------------ | ------------------------- |
| `1`, `true`, `yes`, `on` | `0`, `false`, `no`, `off` |

Anything else warns on stderr and is treated as `false`.

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

### NVS_LOG

**Purpose:** Sets the verbosity of the **developer-facing** log written to stderr. End-user output (the lines a `nvs <subcommand>` user actually reads) is independent of this setting and is governed by the `internal/ui/message` package.

**Default:** `warn`

**Recognized values** (case-insensitive):

| Value             | Shows                    |
| ----------------- | ------------------------ |
| `debug` / `trace` | Everything               |
| `info`            | Info, warn, error, fatal |
| `warn` (default)  | Warn, error, fatal       |
| `error` / `err`   | Error, fatal             |
| `fatal`           | Fatal only               |

**Example:**

```bash
export NVS_LOG=debug       # verbose, useful when filing a bug
export NVS_LOG=error       # silence warnings during scripted runs
```

> [!NOTE]
> `nvs -v` is a shortcut for `NVS_LOG=debug`. The flag and the env var
> are equivalent; when both are set, `NVS_LOG` wins.

---

### NVS_LOG_FILE

**Purpose:** Tee the developer log to a file in addition to stderr. Useful when the terminal UI (spinners, picker panels) would otherwise get in the way of reading the trace.

**Default:** None

**Example:**

```bash
export NVS_LOG=debug
export NVS_LOG_FILE=/tmp/nvs.log
nvs install stable
tail -f /tmp/nvs.log
```

The file is opened in append mode with `0600` permissions. If the file cannot be opened, `nvs` exits with a non-zero status and a clear error — there is no silent fallback to a half-broken logger.

---

## Theming

nvs colors its output through a single nine-slot palette. Every slot is overridable via the `NVS_COLOR_<NAME>` family of environment variables, so you can re-skin the CLI to match your terminal theme without recompiling.

### Default palette: Pastel Twilight

The shipped defaults are the **Pastel Twilight** theme — a deep violet twilight sky lit by soft pastel accents. The "Dark" variant is the theme as designed; the "Light" variant is the same palette darkened so it reads on light backgrounds.

| Slot      | Light (light bg)   | Dark (dark bg)  | Hex (dark) |
| --------- | ------------------ | --------------- | ---------- |
| `PRIMARY` | Darker lavender    | Lavender dreams | `#c9a0e9`  |
| `TEXT`    | Horizon shadows    | Moonlight glow  | `#e0def4`  |
| `MUTED`   | Gentle twilight    | Moonlit clouds  | `#9a96b5`  |
| `SUBTLE`  | Faded midtone      | Muted gray      | `#5a5672`  |
| `BORDER`  | Horizon shadows    | Distant hills   | `#3a364d`  |
| `ACCENT`  | Darker dusk blue   | Dusk blue       | `#80b8e8`  |
| `SUCCESS` | Darker pastel mint | Pastel mint     | `#abe9b3`  |
| `WARNING` | Darker lantern     | Gentle lantern  | `#f9e2af`  |
| `ERROR`   | Darker blush pink  | Blush pink      | `#f28fad`  |

Run `nvs env` (the `Theming` section) to see the resolved values, or `nvs env --json` to consume them from a script.

### Palette slot semantics

| Slot      | Used for                                                    |
| --------- | ----------------------------------------------------------- |
| `PRIMARY` | Brand accent, wordmark, current/active markers              |
| `TEXT`    | Default body text                                           |
| `MUTED`   | Secondary text (descriptions, captions)                     |
| `SUBTLE`  | Tertiary text (timestamps, hints, dimmed labels)            |
| `BORDER`  | Panel/box outline                                           |
| `ACCENT`  | Secondary highlight (paths, commit hashes, version strings) |
| `SUCCESS` | Positive outcomes (install ok, switch ok)                   |
| `WARNING` | Non-fatal issues (already up to date, missing optional dep) |
| `ERROR`   | Fatal outcomes (install failed, permission denied)          |

### Override syntax

For every slot `<NAME>`, three variables control the color:

| Variable                 | Effect                                                         |
| ------------------------ | -------------------------------------------------------------- |
| `NVS_COLOR_<NAME>`       | Sets **both** the light-background and dark-background variant |
| `NVS_COLOR_<NAME>_LIGHT` | Sets only the light-background variant (overrides the base)    |
| `NVS_COLOR_<NAME>_DARK`  | Sets only the dark-background variant (overrides the base)     |

`_LIGHT` and `_DARK` take precedence over the base variable. Precedence (highest first):

1. `NVS_COLOR_<NAME>_LIGHT`
2. `NVS_COLOR_<NAME>_DARK`
3. `NVS_COLOR_<NAME>`

**Examples:**

```bash
# One-liner: paint every nvs prompt in the Catppuccin Mocha mauve.
export NVS_COLOR_PRIMARY="#cba6f7"

# Two-tone: a soft primary in dark mode, deeper primary in light mode.
export NVS_COLOR_PRIMARY_DARK="#cba6f7"
export NVS_COLOR_PRIMARY_LIGHT="#8839ef"

# Use a named color (one of: black, red, green, yellow, blue, magenta, cyan, white, gray, grey).
export NVS_COLOR_SUCCESS="green"

# Use an ANSI 256 number.
export NVS_COLOR_ACCENT="212"
```

### Accepted color formats

Each value is validated. If the value is not a recognized color, nvs warns on stderr and falls back to the default — the whole palette does not break because of one typo.

| Format      | Example                 | Notes                                                                                                             |
| ----------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------- |
| Hex 3-digit | `abc`, `#abc`           | Expands to `aabbcc`. The `#` is optional.                                                                         |
| Hex 6-digit | `abcdef`, `#abcdef`     | Standard RGB. The `#` is optional.                                                                                |
| Hex 8-digit | `abcdef12`, `#abcdef12` | RGBA; the trailing 2 hex digits are the alpha.                                                                    |
| Named       | `red`, `RED`, `Red`     | One of: `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `gray`, `grey` (case-insensitive). |
| ANSI 256    | `0`..`255`              | The xterm 256-color palette.                                                                                      |

A value like `chartreuse` or `#xyz` is rejected, nvs prints `nvs: NVS_COLOR_PRIMARY="..." is not a valid color (...)` once, and the slot keeps its default.

### Picker (huh) follows the palette

The interactive picker (used for `nvs install --pick`, confirmations, etc.) draws its colors from the same palette:

- focused title / selector / button → `Primary.Dark` (i.e. `NVS_COLOR_PRIMARY` / `NVS_COLOR_PRIMARY_DARK`)
- unselected option → `Text.Dark` (i.e. `NVS_COLOR_TEXT` / `NVS_COLOR_TEXT_DARK`)
- blurred title → `Subtle.Dark` (i.e. `NVS_COLOR_SUBTLE` / `NVS_COLOR_SUBTLE_DARK`)
- focused-button foreground (text on the primary background) → `Text.Light` (i.e. `NVS_COLOR_TEXT_LIGHT`)

So there are **no** separate `NVS_PICKER_*` variables: override the corresponding `NVS_COLOR_*` slot and the picker follows automatically.

### Inspecting the active theme

```bash
nvs env              # human-readable table with the current effective values
nvs env --json       # machine-readable (e.g. for dotfiles / theme-sync scripts)
```

Both forms include a `Theming` section listing every `NVS_COLOR_*` slot with its resolved `Light: ..., Dark: ...` value.

### Disabling color entirely

The standard tooling conventions win over every `NVS_COLOR_*` override:

```bash
NO_COLOR=1   nvs install stable    # disable all ANSI escapes
FORCE_COLOR=1 nvs list | less      # force color even when piped
```

See [`NO_COLOR`](https://no-color.org) and [`FORCE_COLOR`](https://force-color.org) for the cross-tool specs.

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

## Logging

**nvs** keeps two distinct output streams:

- **User-facing** — the short, always-on status/error messages the user reads. These go to stdout (in `--json` mode) or stderr (default). They are styled by the same palette as the rest of the CLI.
- **Developer-facing** — structured key/value traces that help a developer (or a bug filer) follow what `nvs` is doing internally. These go to **stderr** by default at `warn` level. They are off by default so a normal `nvs <subcommand>` does not flood the terminal.

| Stream           | Package               | Default level | Toggle                            |
| ---------------- | --------------------- | ------------- | --------------------------------- |
| User-facing      | `internal/ui/message` | always on     | (no toggle)                       |
| Developer-facing | `internal/log`        | `warn`        | `-v` / `NVS_LOG` / `NVS_LOG_FILE` |

To turn on the developer log for a single command:

```bash
nvs -v install stable
NVS_LOG=debug nvs install stable
```

To capture it to a file (useful when the terminal UI hides the traces):

```bash
NVS_LOG=debug NVS_LOG_FILE=/tmp/nvs.log nvs install stable
```

See [NVS_LOG](#nvs_log) and [NVS_LOG_FILE](#nvs_log_file) for the full reference.

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
