# Installation Guide

This guide covers all available methods to install **nvs** on your system.

## Prerequisites

- **Operating System:** macOS (Intel & Apple Silicon), Linux, or Windows
- **Permissions:** Ability to create symlinks (may require administrator privileges on Windows)

> [!warning]
> To ensure a smooth experience, remove any existing Neovim installations from your system and let **nvs** manage them instead.

## Installation Methods

### Method 1: Install Script (Recommended)

The easiest way to install **nvs** is using our installation script, which automatically detects your operating system and architecture.

> [!warning]
> Always review remote scripts before execution. Inspect the script contents at [install.sh](https://github.com/y3owk1n/nvs/blob/main/install.sh) to ensure safety.

```bash
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/install.sh | bash
```

> [!note]
> You can upgrade **nvs** by running the same installation script again.

#### Uninstalling

If you need to uninstall **nvs**, use the provided uninstall script:

```bash
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/uninstall.sh | bash
```

### Method 2: Homebrew (macOS/Linux)

Install **nvs** via Homebrew:

```bash
brew tap y3owk1n/tap
brew install y3owk1n/tap/nvs
```

### Method 3: Pre-built Binaries

1. Download the [latest release binary](https://github.com/y3owk1n/nvs/releases) for your system
2. Make the binary available in your system's PATH
3. Ensure it's executable: `chmod +x nvs`

### Method 4: Nix (Flakes)

If you're using Nix with flakes enabled, you can integrate **nvs** into your Nix configuration. The Nix packages include shell completions for bash, zsh, and fish.

> [!note]
> When installing via Nix, shell completions are automatically configured for supported shells.

#### Using Overlay (Recommended)

Add the **nvs** overlay to your Nix configuration and use it as a regular package:

```nix
{
  inputs = {
    nvs.url = "github:y3owk1n/nvs";
  };

  outputs = { nvs, ... }: {
    nixosConfigurations."your-host" = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        {
          nixpkgs.overlays = [ nvs.overlays.default ];
          environment.systemPackages = [
            pkgs.nvs        # Latest prebuilt
            # or
            pkgs.nvs-source # Build from source
          ];
        }
      ];
    };
  };
}
```

#### Using Home Manager

Add **nvs** to your `home.nix`:

```nix
{
  inputs = {
    nvs.url = "github:y3owk1n/nvs";
  };

  outputs = { nvs, ... }: {
    home.packages = [
      nvs.packages.${pkgs.system}.default  # Latest prebuilt
      # or
      nvs.packages.${pkgs.system}.source   # Build from source
    ];
  };
}
```

#### Using nix-darwin (macOS)

Add **nvs** to your `darwin-configuration.nix`:

```nix
{
  inputs = {
    nvs.url = "github:y3owk1n/nvs";
  };

  outputs = { nvs, ... }: {
    darwinConfigurations."your-host" = nix-darwin.lib.darwinSystem {
      modules = [
        {
          nixpkgs.overlays = [ nvs.overlays.default ];
          environment.systemPackages = [
            pkgs.nvs        # Latest prebuilt
            # or
            pkgs.nvs-source # Build from source
          ];
        }
      ];
    };
  };
}
```

#### Direct Installation (not recommended for Nix environments)

For temporary use or testing:

```bash
# Run latest version
nix run github:y3owk1n/nvs

# Run source version
nix run github:y3owk1n/nvs#source
```

### Method 5: Build from Source

If you prefer to build from source, ensure you have [Go](https://golang.org/dl/) (v1.25 or later) installed.

```bash
git clone https://github.com/y3owk1n/nvs.git
cd nvs
go build -o nvs ./main.go
```

For cross-platform builds:

```bash
# macOS ARM64
env GOOS=darwin GOARCH=arm64 go build -o nvs-darwin-arm64 ./main.go

# macOS Intel
env GOOS=darwin GOARCH=amd64 go build -o nvs-darwin-amd64 ./main.go

# Linux ARM64
env GOOS=linux GOARCH=arm64 go build -o nvs-linux-arm64 ./main.go

# Linux AMD64
env GOOS=linux GOARCH=amd64 go build -o nvs-linux-amd64 ./main.go

# Windows
env GOOS=windows GOARCH=amd64 go build -o nvs-windows-amd64.exe ./main.go
```

Move the built binary to a directory in your PATH.

## Post-Installation Setup

After installing **nvs**, you need to configure your environment. See the [Configuration Guide](CONFIGURATION.md) for detailed instructions.

## Verification

After installation and setup, verify **nvs** is working:

```bash
nvs --version
nvs --help
```

## Troubleshooting

### Windows Issues

**nvs** has limited Windows support. If you encounter issues:

- Ensure you're running in an administrator terminal for symlink creation
- Check that your antivirus isn't blocking the binary
- Verify PATH environment variables are set correctly

### Permission Issues

If you encounter permission errors:

- On Unix systems: Ensure you have write permissions to `~/.local/bin` (or your custom `NVS_BIN_DIR`)
- On Windows: Run your terminal as administrator

### Build Dependencies

When building Neovim from commits, ensure these tools are installed:

- `git`
- `make`
- `cmake`

On macOS, also install:

```bash
brew install ninja cmake gettext curl
```

## Next Steps

Once installed, proceed to:

- [Configure your environment](CONFIGURATION.md)
- [Learn basic usage](USAGE.md)

For development setup, see the [Development Guide](DEVELOPMENT.md).

## Related Documentation

- [Usage Guide](USAGE.md) - Command reference
- [Configuration Guide](CONFIGURATION.md) - Environment setup
- [Contributing Guide](CONTRIBUTING.md) - For contributors
