# Installation Guide

This guide covers all available methods to install **nvs** on your system.

## Prerequisites

- **Operating System:** macOS (Intel & Apple Silicon), Linux, or Windows (PowerShell required)
- **Permissions:** Ability to create symlinks (may require administrator privileges on Windows)

> [!warning]
> To ensure a smooth experience, remove any existing Neovim installations from your system and let **nvs** manage them instead.

## Installation Methods

### Method 1: Install Script (Recommended)

The easiest way to install **nvs** is using our installation script, which automatically detects your operating system and architecture.

> [!warning]
> Always review remote scripts before execution. Inspect the script contents at [install.sh](https://github.com/y3owk1n/nvs/blob/main/install.sh) to ensure safety.

#### Unix-like Systems (Linux, macOS, WSL)

```bash
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/install.sh | bash
```

#### Windows

> [!warning]
> Windows support is not fully tested as the author does not use Windows. Please report any issues and feel free to contribute improvements.

For Windows users, use the PowerShell installation script:

```powershell
# Download and run the installer
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/y3owk1n/nvs/main/install.ps1" -OutFile "install.ps1"

# If you get an execution policy error, run:
# Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned

.\install.ps1
```

> [!note]
> You can upgrade **nvs** by running the same installation script again.

#### Uninstalling

If you need to uninstall **nvs**, use the provided uninstall script:

**Unix-like Systems (Linux, macOS, WSL):**

```bash
curl -fsSL https://raw.githubusercontent.com/y3owk1n/nvs/main/uninstall.sh | bash
```

**Windows:**

> [!warning]
> Windows support is not fully tested as the author does not use Windows. Please report any issues and feel free to contribute improvements.

```powershell
# Download and run the uninstaller
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/y3owk1n/nvs/main/uninstall.ps1" -OutFile "uninstall.ps1"

# If you get an execution policy error, run:
# Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned

.\uninstall.ps1
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

#### Add Flake Input

Add the **nvs** flake input to your Nix configuration:

```nix
{
  inputs = {
    nvs.url = "github:y3owk1n/nvs";
  };
}
```

#### Home-manager Module

Use the home-manager module for user-specific installation:

```nix
{
  outputs = { self, nixpkgs, home-manager, nvs, ... }: {
    homeConfigurations.your-username = home-manager.lib.homeManagerConfiguration {
      pkgs = nixpkgs.legacyPackages.aarch64-darwin;

      modules = [
        # Apply the nvs overlay
        {
          nixpkgs.overlays = [ nvs.overlays.default ];
        }

        # Import the nvs module
        nvs.homeManagerModules.default

        # Configure nvs
        {
          # Enable nvs
          programs.nvs.enable = true;

          # Optional: Use specific package version
          # programs.nvs.package = pkgs.nvs; # This will use the latest version
          # programs.nvs.package = pkgs.nvs-source; # This will build from source

          # Optional: Customize nvs settings
          # programs.nvs = {
          #   enable = true;
          #   enableAutoSwitch = true;  # Enable automatic version switching
          #   enableShellIntegration = true;  # Enable shell integration
          #   configDir = "${config.xdg.configHome}/nvs";
          #   cacheDir = "${config.xdg.cacheHome}/nvs";
          #   binDir = "${config.home.homeDirectory}/.local/bin";
          #   shellIntegration = {
          #     bash = true;
          #     zsh = true;
          #     fish = true;
          #   };
          # };
        }
      ];
    };
  };
}
```

The Home Manager module automatically:

- Installs nvs
- Sets up environment variables (`NVS_CONFIG_DIR`, `NVS_CACHE_DIR`, `NVS_BIN_DIR`)
- Adds the binary directory to your PATH
- Enables shell integration for automatic environment setup
- Enables automatic version switching when entering directories with `.nvs-version` files
- Creates necessary directories

#### Using Overlay Only

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

**nvs** supports Windows through native PowerShell scripts. If you encounter issues:

- Ensure you're running PowerShell with appropriate permissions (no admin required for user installation)
- Check that your antivirus isn't blocking the binary download
- Verify PATH environment variables are set correctly (restart terminal after installation)
- The installer automatically adds **nvs** to your user PATH

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
