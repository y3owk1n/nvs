# nvs (Neovim Version Switcher)

**Neovim Version Switcher** – Easily install, switch, and manage multiple versions and config of Neovim like a boss 🚀

[![GitHub release](https://img.shields.io/github/release/y3owk1n/nvs.svg)](https://github.com/y3owk1n/nvs/releases) [![License](https://img.shields.io/github/license/y3owk1n/nvs.svg)](LICENSE)

## 👀 Overview

**nvs** (Neovim Version Switcher/Manager) is a lightweight cross-platform CLI tool written in Go 🏗️ that makes it super easy to install, switch between, and manage multiple versions of Neovim and config on your machine. Whether you’re testing a cutting‑edge nightly build 🌙 or sticking with the stable release 🔒, **nvs** has got your back!

> [!note]
> I only have a mac and it's working perfectly fine for my use case. If it's not working for other OS, feel free to help fixing that or share it as an issue. I'll try to look into it.

## 🌟 Showcase

<https://github.com/user-attachments/assets/df307bfa-058d-4b23-b4bf-a23bdcba458f>

## 🌟 Features

- **Easy Installation:**
  Download and install Neovim versions directly from GitHub with a single command.
- **Version Switching:**
  Switch between installed versions in a snap. **nvs** updates a global symlink so your preferred version is always just a command away.
- **Config Switching:**
  Easily toggle between Neovim configurations by scanning ~/.config (including symlinks) and setting NVIM_APPNAME interactively or via a direct subcommand argument.
- **Remote Version Listing:**
  List all available remote releases (stable, nightly, etc.) with cached results to avoid GitHub rate limits ⚡. Need fresh data? Just add the `force` flag.
- **Upgrade for Stable and Nightly:**
  Easily upgrade your installed stable and/or nightly versions. The upgrade command checks if you’re already on the latest version and only performs an upgrade if needed.
- **Uninstallation & Reset:**
  Remove individual versions or reset your entire configuration with ease. (Full cleanup? See the caveats! ⚠️)
- **Cross-Platform:**
  Works on macOS (Intel & Apple Silicon), Linux, and Windows.
- **Global Symlink Management:**
  Automatically creates a consistent global binary in `~/.nvs/bin` for a seamless experience.

## 🚀 Installation

### Install with `install.sh`

You can install **nvs** with a single command that downloads and executes our installation script. The script automatically detects your operating system and architecture and installs the appropriate binary.

> [!warning]
> Always review remote scripts before execution. Before running any script from the internet, inspect its contents to ensure its safety.

```bash
# Using curl
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/install.sh | bash

# Using wget
wget -qO- https://raw.githubusercontent.com/y3owk1n/nvs/main/install.sh | bash
```

We have also included an `uninstall script` if you would like to uninstall it

```bash
# Using curl
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/uninstall.sh | bash

# Using wget
wget -qO- https://raw.githubusercontent.com/y3owk1n/nvs/main/uninstall.sh | bash
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
````

Move the binary to your PATH or run it directly.

## 💻 Usage

**nvs** uses a clean subcommand interface. Run `nvs --help` for full details.

> [!note]
> Remember to add `nvs` to your path. See next section about how.

### Commands

#### path

Best effort to automatically add the global binary directory to your PATH.

```bash
nvs path
```

#### install

Install a specific Neovim version.

```bash
nvs install stable    # Install the latest stable release 🔒
nvs install nightly   # Install the latest nightly release 🌙
nvs install v0.10.3   # Install a specific version
nvs install 0.10.3    # Or without the v keyword

# or with shorthand
nvs i stable
```

#### use

Switch to a particular version. This updates a global symlink in ~/.nvs/bin so that you can simply run nvim.

```bash
nvs use stable
nvs use nightly
nvs use v0.10.3
nvs use 0.10.3
```

> [!warning]
> If you're using windows, I think you will need `administrator` privilege terminal for **nvs** to symlink.

#### list

List available remote releases and installed status (cached for 5 minutes to avoid rate limiting). Use the force flag to refresh the cache.

> [!note]
> The list will be filtered out for only nightly, stable, all version that are above v0.5.0.

```bash
nvs list
nvs list force

# or with shorthand
nvs ls
nvs ls force
```

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

> [!note]
> It only scan the `~/.config` directory with names contain `nvim` in it. E.g. nvim, nvim-test, nvim-vanilla, ...

```bash
nvs config
nvs config nvim-test

# or with shorthand
nvs c
nvs conf
```

#### uninstall

Uninstall an installed version.

```bash
nvs uninstall stable
nvs uninstall nightly
nvs uninstall v0.10.3
nvs uninstall 0.10.3

# or with shorthand
nvs rm stable
nvs remove nightly
nvs un 0.10.3
```

#### reset

Reset to factory state.

> [!warning]
> This command will delete all data in ~/.nvs including items inside the bin directory, but will preserve the bin directory structure. Use with caution.

```bash
nvs reset
```

## 🔗 Adding **nvs** to Your PATH

To easily run the Neovim binary provided by **nvs**, you need to add the global bin directory (`~/.nvs/bin`) to your PATH. Below are instructions for common shells:

> [!note]
> We have provided `nvs path` command for the best effort to automatically setup the path for you in common shells. If it does not work, you need to set it up manually.

### Macos Or Linux

#### Bash

Add the following line to your `~/.bashrc` (or `~/.bash_profile` on macOS):

```bash
export PATH="$HOME/.nvs/bin:$PATH"
```

Then, reload your configuration:

```bash
source ~/.bashrc   # or source ~/.bash_profile
```

#### Zsh

Add the following line to your `~/.zshrc`:

```bash
export PATH="$HOME/.nvs/bin:$PATH"
```

Then, reload your configuration:

```bash
source ~/.zshrc
```

#### Fish

Add the following line to your `~/.config/fish/config.fish`:

```bash
set -gx PATH $HOME/.nvs/bin $PATH
```

Then, reload your configuration:

```bash
source ~/.config/fish/config.fish
```

### Windows

Open an elevated Command Prompt (Run as administrator) and type:

```bash
setx PATH "%PATH%;C:\Users\YourName\.nvs\bin"
```

You may need to open a new Command Prompt session to see the updated PATH and try running `nvim`.

## 🧩 Shell Completions

**nvs** supports generating shell completions using Cobra’s built‐in functionality. You can easily enable command completions for your favorite shell by following the instructions below.

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

## 📂 Configuration & Data

**nvs** stores its configuration, downloaded versions, and cache in the ~/.nvs directory.

> [!note]
> Remember: Homebrew will not delete this directory upon uninstallation, you must delete it manually if you want a full cleanup.

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
