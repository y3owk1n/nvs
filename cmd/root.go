package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// verbose controls the log level.
	verbose bool

	// ctx is the global context used by the CLI.
	// cancel cancels the context, e.g. on interrupt signals.
	ctx, cancel = context.WithCancel(context.Background())

	// versionsDir is the directory where installed Neovim versions are stored.
	versionsDir string

	// cacheFilePath is the path to the file that caches remote release data.
	cacheFilePath string

	// globalBinDir is the directory where the global nvim symlink is created.
	globalBinDir string

	// Version of nvs, defaults to "v0.0.0" but may be set during build time.
	Version = "v0.0.0"
)

// Execute initializes the configuration, sets up global flags, and executes the root command.
// Example usage:
//
//	func main() {
//	    cmd.Execute()
//	}
func Execute() {
	// Initialize configuration before running any commands.
	cobra.OnInitialize(initConfig)

	// Set a persistent flag for verbose logging.
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	// Execute the root command with the global context.
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

// initConfig is called automatically on command initialization.
// It sets up logging levels, handles OS signals for graceful shutdown, and ensures that necessary
// directories (config, versions, cache, binary) exist, using environment variables as overrides when available.
//
// Example behavior:
//   - If NVS_CONFIG_DIR is set, it is used as the config directory; otherwise, the system config directory is used.
//   - Similar logic applies for cache (NVS_CACHE_DIR) and binary directories (NVS_BIN_DIR).
func initConfig() {
	// Set logging level based on the verbose flag.
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Verbose mode enabled")
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Set up a signal handler to cancel the global context on an interrupt signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println()
		logrus.Debug("Interrupt received, canceling operations...")
		cancel()
		os.Exit(1)
	}()

	// Determine the base configuration directory.
	var baseConfigDir string
	if custom := os.Getenv("NVS_CONFIG_DIR"); custom != "" {
		baseConfigDir = custom
		logrus.Debugf("Using custom config directory from NVS_CONFIG_DIR: %s", baseConfigDir)
	} else {
		if configDir, err := os.UserConfigDir(); err == nil {
			baseConfigDir = filepath.Join(configDir, "nvs")
			logrus.Debugf("Using system config directory: %s", baseConfigDir)
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				logrus.Fatalf("Failed to get user home directory: %v", err)
			}
			baseConfigDir = filepath.Join(home, ".nvs")
			logrus.Debugf("Falling back to home directory for config: %s", baseConfigDir)
		}
	}

	// Ensure the configuration directory exists.
	if err := os.MkdirAll(baseConfigDir, 0755); err != nil {
		logrus.Fatalf("Failed to create config directory: %v", err)
	}
	logrus.Debugf("Config directory ensured: %s", baseConfigDir)

	// Set the directory for installed versions.
	versionsDir = filepath.Join(baseConfigDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		logrus.Fatalf("Failed to create versions directory: %v", err)
	}
	logrus.Debugf("Versions directory ensured: %s", versionsDir)

	// Determine the base cache directory.
	var baseCacheDir string
	if custom := os.Getenv("NVS_CACHE_DIR"); custom != "" {
		baseCacheDir = custom
		logrus.Debugf("Using custom cache directory from NVS_CACHE_DIR: %s", baseCacheDir)
	} else {
		if cacheDir, err := os.UserCacheDir(); err == nil {
			baseCacheDir = filepath.Join(cacheDir, "nvs")
			logrus.Debugf("Using system cache directory: %s", baseCacheDir)
		} else {
			baseCacheDir = filepath.Join(baseConfigDir, "cache")
			logrus.Debugf("Falling back to config directory for cache: %s", baseCacheDir)
		}
	}
	// Ensure the cache directory exists.
	if err := os.MkdirAll(baseCacheDir, 0755); err != nil {
		logrus.Fatalf("Failed to create cache directory: %v", err)
	}
	cacheFilePath = filepath.Join(baseCacheDir, "releases.json")
	logrus.Debugf("Cache directory ensured: %s", baseCacheDir)
	logrus.Debugf("Cache file path set: %s", cacheFilePath)

	// Determine the base binary directory.
	var baseBinDir string
	if custom := os.Getenv("NVS_BIN_DIR"); custom != "" {
		baseBinDir = custom
		logrus.Debugf("Using custom binary directory from NVS_BIN_DIR: %s", baseBinDir)
	} else {
		if runtime.GOOS == "windows" {
			home, err := os.UserHomeDir()
			if err != nil {
				logrus.Fatalf("Failed to get user home directory: %v", err)
			}
			baseBinDir = filepath.Join(home, "AppData", "Local", "Microsoft", "WindowsApps")
			logrus.Debugf("Using Windows binary directory: %s", baseBinDir)
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				logrus.Fatalf("Failed to get user home directory: %v", err)
			}
			baseBinDir = filepath.Join(home, ".local", "bin")
			logrus.Debugf("Using default binary directory: %s", baseBinDir)
		}
	}
	// Ensure the binary directory exists.
	if err := os.MkdirAll(baseBinDir, 0755); err != nil {
		logrus.Fatalf("Failed to create binary directory: %v", err)
	}
	globalBinDir = baseBinDir
	logrus.Debugf("Global binary directory ensured: %s", globalBinDir)
}

// rootCmd is the base command for the CLI.
// It holds the main description and version of the tool.
// This command is the entry point for subcommands such as "install", "list", "reset", etc.
//
// Example usage (from command-line):
//
//	nvs --help
var rootCmd = &cobra.Command{
	Use:     "nvs",
	Short:   "Neovim version switcher",
	Long:    "A CLI tool to install, switch, list, uninstall, and reset Neovim versions.",
	Version: Version,
}
