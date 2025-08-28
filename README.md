# nvs (Neovim Version Switcher)

**Neovim Version Switcher** – Easily install, switch, and manage multiple versions (including commit hashes) and config of Neovim like a boss 🚀

[![CI](https://github.com/y3owk1n/nvs/actions/workflows/ci.yml/badge.svg)](https://github.com/y3owk1n/nvs/actions/workflows/ci.yml) [![GitHub release](https://img.shields.io/github/release/y3owk1n/nvs.svg)](https://github.com/y3owk1n/nvs/releases) [![License](https://img.shields.io/github/license/y3owk1n/nvs.svg)](LICENSE)

## 👀 Overview

**nvs** (Neovim Version Switcher/Manager) is a lightweight cross-platform CLI tool written in Go 🏗️ that makes it super easy to install, switch between, and manage multiple versions of Neovim and config on your machine. Whether you’re testing a cutting‑edge nightly build 🌙 or sticking with the stable release 🔒, **nvs** has got your back!

```bash
$ nvs use stable
✓ Switched to Neovim stable

$ nvim -v
NVIM v0.10.4
Build type: Release
LuaJIT 2.1.1713484068
Run "nvim -V1 -v" for more info

$ nvs use nightly
✓ Switched to Neovim nightly

$ nvim -v
NVIM v0.11.0-dev-1961+g7e2b75760f
Build type: RelWithDebInfo
LuaJIT 2.1.1741571767
Run "nvim -V1 -v" for more info

$ nvs install 0.10.0
ℹ Resolving version v0.10.0...
ℹ Installing Neovim v0.10.0...
✓ Installation successful!

$ nvs use 0.10.0
✓ Switched to Neovim v0.10.0

$ nvim -v
NVIM v0.10.0
Build type: Release
LuaJIT 2.1.1713484068
Run "nvim -V1 -v" for more info
```

## 🌟 Showcase

<https://github.com/user-attachments/assets/54f30265-3502-4469-bf5b-f405899e602b>

## 🌟 Features

- **Easy Installation:**
  Download and install Neovim versions directly from GitHub with a single command.
- **Version Switching:**
  Switch between installed versions in a snap. **nvs** updates a global symlink so your preferred version is always just a command away.
- **Config Switching:**
  Easily toggle between Neovim configurations by scanning `~/.config` (including symlinks) and setting `NVIM_APPNAME` interactively or via a direct subcommand argument.
- **Remote Version Listing:**
  List all available remote releases (stable, nightly, etc.) with cached results to avoid GitHub rate limits ⚡. Need fresh data? Just add the `force` flag.
- **Upgrade for Stable and Nightly:**
  Easily upgrade your installed stable and/or nightly versions. The upgrade command checks if you’re already on the latest version and only performs an upgrade if needed.
- **Uninstallation & Reset:**
  Remove individual versions or reset your entire configuration with ease.
- **Cross-Platform:**
  Works on macOS (Intel & Apple Silicon), Linux, and Windows.
- **Verbose Logging:**
  Run with the `--verbose` flag to see detailed logs during directory initialization, reset, and other operations.

## 🚀 Installation

> [!warning]
> To make the process butter smooth, remember to remove any `Neovim` installation in your system and let **nvs** manage them instead.

### Using our install script

You can install **nvs** with a single command that downloads and executes our [installation script](https://github.com/y3owk1n/nvs/blob/main/install.sh). The script automatically detects your operating system and architecture and installs the appropriate binary.

> [!warning]
> Always review remote scripts before execution. Before running any script from the internet, inspect its contents to ensure its safety.

```bash
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/install.sh | bash
```

> [!note]
> You can upgrade by just running the same installation script again.

We have also included an `uninstall script` if you would like to uninstall it

```bash
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/uninstall.sh | bash
```

### Homebrew

Install **nvs** via Homebrew! Simply add our tap:

```bash
brew tap y3owk1n/tap
```

Then install with:

```bash
brew install y3owk1n/tap/nvs
```

### Using a released binary

- Download the [latest release binary](https://github.com/y3owk1n/nvs/releases) for your system
- Make it available globally on PATH environment variable
- Run it like any other method

### Building From Source

Make sure you have [Go](https://golang.org/dl/) (v1.23 or later) installed. Then run:

```bash
git clone https://github.com/y3owk1n/nvs.git
cd nvs
mkdir -p build

# Build for darwin-arm64
env GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-darwin-arm64 ./main.go

# Build for darwin-amd64
env GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-darwin-amd64 ./main.go

# Build for linux-arm64
env GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-linux-arm64 ./main.go

# Build for linux-amd64
env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-linux-amd64 ./main.go

# Build for windows-amd64
env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-windows64.exe ./main.go
```

Move the binary to your PATH or run it directly.

## 🌈 Automatic Configuration

Environment variables need to be setup before you can start using nvs.
This is done by evaluating the output of `nvs env --source`.

`--source` mode detects your current shell (`bash`, `zsh`, `fish`, etc.) and prints the correct syntax to set `NVS_CONFIG_DIR`, `NVS_CACHE_DIR`, `NVS_BIN_DIR`, and update your `PATH`.

If auto-detection fails, you can explicitly specify the shell type:

```bash
nvs env --source --shell bash
nvs env --source --shell zsh
nvs env --source --shell fish
```

> [!note]
> If you install `nvs` on multiple machines or inside different environments, using --shell explicitly can avoid relying on shell detection:

##### Bash

Add the following to your `.bashrc` profile:

```bash
eval "$(nvs env --source)"
```

##### Zsh

Add the following to your `.zshrc` profile:

```zsh
eval "$(nvs env --source)"
```

##### Fish shell

Create `~/.config/fish/conf.d/nvs.fish` and add this line to it:

```fish
nvs env --source | source
```

## 📂 Configuration & Data

**nvs** now follows OS-specific best practices for storing configuration, cache, and binary data. By default:

- **Configuration Files:**
  Stored in `~/.config/nvs` on Unix-like systems (or `AppData\Roaming\nvs` on Windows).
  Override with the `NVS_CONFIG_DIR` environment variable.
- **Cache Files:**
  Stored in `~/.cache/nvs` on Unix-like systems (or `AppData\Local\nvs` on Windows).
  Override with the `NVS_CACHE_DIR` environment variable.
- **Global Binary Symlinks:**
  Placed in `~/.local/bin` on Unix-like systems (or `AppData\Local\Programs` on Windows).
  Override with the `NVS_BIN_DIR` environment variable.

> [!note]
> All these directories will be created upon running any command (including `nvs --help`) the first time after installation.
> If you set custom paths after that, feel free to manually delete the default directories.

### Overriding Default Directories with Environment Variables

**nvs** allows you to customize the locations where configuration, cache, and binary files are stored by setting the following environment variables:

- `NVS_CONFIG_DIR` Overrides the default configuration directory.
  - Default:
    - On Unix-like systems: `~/.config/nvs`
    - On Windows: `AppData\Roaming\nvs`
- `NVS_CACHE_DIR` Overrides the default cache directory.
  - Default:
    - On Unix-like systems: `~/.cache/nvs`
    - On Windows: `AppData\Local\nvs`
- `NVS_BIN_DIR` Overrides the default global binary directory.
  - Default:
    - On Unix-like systems: `~/.local/bin`
    - On Windows: `AppData\Local\Programs`

#### How to Set These Environment Variables Manually

##### On Unix-like Systems (Linux/macOS)

You can add the following lines to your shell configuration file (e.g., `~/.bashrc`, `~/.zshrc`):

```bash
# Override nvs directories
export NVS_CONFIG_DIR="$HOME/custom-config/nvs"
export NVS_CACHE_DIR="$HOME/custom-cache/nvs"
export NVS_BIN_DIR="$HOME/custom-bin"
```

After editing your configuration file, reload it:

```bash
source ~/.bashrc   # or source ~/.zshrc
```

##### On Windows

You can set environment variables temporarily in the Command Prompt:

```bash
set NVS_CONFIG_DIR=C:\Path\To\Custom\Config
set NVS_CACHE_DIR=C:\Path\To\Custom\Cache
set NVS_BIN_DIR=C:\Path\To\Custom\Bin
```

For a permanent change, open the **System Properties** → **Advanced** → **Environment Variables** and add or edit the variables there. Alternatively, you can use the `setx` command in an elevated Command Prompt:

```bash
setx NVS_CONFIG_DIR "C:\Path\To\Custom\Config"
setx NVS_CACHE_DIR "C:\Path\To\Custom\Cache"
setx NVS_BIN_DIR "C:\Path\To\Custom\Bin"
```

> [!note]
> If you override the binary directory (`NVS_BIN_DIR`), make sure to update your system's PATH variable accordingly so that the **nvs** binaries can be found by your shell.

## 💻 Usage

**nvs** uses a clean subcommand interface. Run `nvs --help` for full details.

### Commands

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

#### list-remote

List available remote releases and installed status (cached for 5 minutes to avoid rate limiting). Use the force flag to refresh the cache.

> [!note]
> The list will be filtered out for only nightly, stable, and all version that are above v0.5.0.

```bash
nvs list-remote
nvs list-remote force # force refresh the cache

# or with shorthand
nvs ls-remote
nvs ls-remote force
```

#### list

List all locally installed Neovim versions

```bash
nvs list

# or with shorthand
nvs ls
```

#### install

Install a specific Neovim version or build from commits.

> [!warning]
> To build from commit, make sure your system has `git`, `make`, and `cmake` installed, as building Neovim requires these 3 pieces to work together.

> [!warning]
> If you're using macos make sure to also run `brew install ninja cmake gettext curl`.

```bash
nvs install stable    # Install the latest stable release 🔒
nvs install nightly   # Install the latest nightly release 🌙
nvs install v0.10.3   # Install a specific version
nvs install 0.10.3    # Or without the v keyword
nvs install master    # Build and install based on the latest master commit
nvs install 2db1ae3   # Build and install a specific commit hash in 7 character
nvs install 2db1ae37f14d71d1391110fe18709329263c77c9 # Or a 40 character hash

# or with shorthand
nvs i stable
```

#### use

Switch to a particular version. This updates a global symlink in your designated bin directory so that you can simply run nvim.

> [!note] > **nvs** will attempt to install the version specified if it's not installed and switch to it later.

```bash
nvs use stable
nvs use nightly
nvs use v0.10.3
nvs use 0.10.3
nvs use 2db1ae3
```

> [!warning]
> If you're using windows, I think you will need `administrator` privilege terminal for **nvs** to symlink.

#### current

Display the currently active Neovim version.

```bash
nvs current
```

#### upgrade

Upgrade installed stable and/or nightly versions. If no argument is provided, both stable and nightly are upgraded (if installed).

> [!note]
> The upgrade command checks if the installed release is already up-to-date by comparing a stored identifier (release tag for stable, commit hash or published date for nightly). If no upgrade is needed, you'll be informed that you're on the latest version.

```bash
nvs upgrade         # Upgrades both stable and nightly if installed
nvs upgrade stable  # Upgrades only the stable release if installed
nvs upgrade nightly # Upgrades only the nightly release if installed

# or with shorthand
nvs up
nvs up stable
nvs up nightly
```

#### config

Switch between multiple configs. If no argument is provided, it will promp a select UI, else it will just open with specified name.

> [!note] > **nvs** scans your configuration directory (e.g. `~/.config`) for entries containing `nvim` in the name. Examples includes `nvim`, `nvim-test`, or `nvim-vanilla`.

```bash
nvs config
nvs config nvim-test

# or with shorthand
nvs c
nvs conf
```

#### uninstall

Uninstall an installed version. **nvs** will prompt a confirmation `y/N` if trying to uninstall the current used version and also prompt to select one of installed versions and switch to it.

```bash
nvs uninstall stable
nvs uninstall nightly
nvs uninstall v0.10.3
nvs uninstall 0.10.3
nvs uninstall 2db1ae3

# or with shorthand
nvs rm stable
nvs remove nightly
nvs un 0.10.3
```

#### path

Best effort to automatically add the global binary directory to your PATH.

```bash
nvs path
```

> [!note]
> This won't work if you're managing your system with nix home manager. [See details below how to configure it](#nix-home-manager)

#### env

Prints the env configuration used by NVS (`NVS_CONFIG_DIR`, `NVS_CACHE_DIR`, and `NVS_BIN_DIR`).

```bash
nvs env
```

#### reset

Reset to factory state.

> [!warning]
> This command will delete all data in your configuration and cache directories—determined based on OS conventions or the environment variables (`NVS_CONFIG_DIR` and `NVS_CACHE_DIR`)—and remove the `nvim` binary from your binary directory (or `NVS_BIN_DIR`). Use with caution.

```bash
nvs reset
```

## 🔗 Adding **nvs** to Your PATH

To easily run the Neovim binary provided by nvs, you need to add the global binary directory to your PATH. By default, this is set according to OS best practices and environment variables. On Unix-like systems, the default is:

- **Configuration Directory:** `~/.config/nvs` (or overridden via `NVS_CONFIG_DIR`)
- **Cache Directory:** `~/.cache/nvs` (or overridden via `NVS_CACHE_DIR`)
- **Global Binary Directory:** `~/.local/bin` (or overridden via `NVS_BIN_DIR`)

> [!note]
> We have provided `nvs path` command for the best effort to automatically setup the path for you in common shells. If it does not work, you need to set it up manually. See below with some example.

### Macos Or Linux

#### Bash

Add the following line to your `~/.bashrc` (or `~/.bash_profile` on macOS):

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then, reload your configuration:

```bash
source ~/.bashrc   # or source ~/.bash_profile
```

#### Zsh

Add the following line to your `~/.zshrc`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then, reload your configuration:

```bash
source ~/.zshrc
```

#### Fish

Add the following line to your `~/.config/fish/config.fish`:

```bash
set -gx PATH $HOME/.local/bin $PATH
```

Then, reload your configuration:

```bash
source ~/.config/fish/config.fish
```

### Windows

Open an elevated Command Prompt (Run as administrator) and type:

```bash
setx PATH "%PATH%;%APPDATA%\nvs\bin"
```

Restart your Command Prompt (or log off and back on) for the changes to take effect.

> [!note]
> If you have overridden the default binary directory with the `NVS_BIN_DIR` environment variable, make sure to replace `%APPDATA%\nvs\bin` with your custom path in the above command.

### Nix Home Manager

If you're using home manager with nix, you can try the following as documented at [Nix Home Manager Docs](https://nix-community.github.io/home-manager/options.xhtml#opt-home.sessionPath) :

```nix

{
 home.sessionPath = [
  "$HOME/.local/bin"
 ];
}
```

## 🧩 Shell Completions

**nvs** supports generating shell completions using Cobra’s built‐in functionality. You can easily enable command completions for your favorite shell by following the instructions below.

```bash
Generate the autocompletion script for nvs for the specified shell.
See each sub-command's help for details on how to use the generated script.

Usage:
  nvs completion [command]

Available Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh
```

### Bash

To enable Bash completions, add the following line to your `~/.bashrc` (or `~/.bash_profile` on macOS):

```bash
source <(nvs completion bash)
```

Then, reload your configuration:

```bash
source ~/.bashrc   # or source ~/.bash_profile
```

### Zsh

For Zsh users, first ensure that completion is enabled by adding the following to your ~/.zshrc (if not already present):

```bash
autoload -U compinit && compinit
```

Then add the following line to generate and load **nvs** completions:

```bash
source <(nvs completion zsh)
```

Then, reload your configuration:

```bash
source ~/.zshrc
```

### Fish

Fish shell users can generate completions with:

```bash
nvs completion fish | source
```

To make the completions permanent, save them to your completions directory:

```bash
nvs completion fish > ~/.config/fish/completions/nvs.fish
```

Then, reload your configuration:

```bash
source ~/.config/fish/config.fish
```

## 🤝 Contributing

Contributions are always welcome! Here's how you can help:

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to your branch
5. Open a pull request

## 📄 License

This project is licensed under the MIT License. Feel free to use, modify, and distribute it as you see fit.

Enjoy using **nvs**, and may your Neovim sessions be ever lit! ✨
