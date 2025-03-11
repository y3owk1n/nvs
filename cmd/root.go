package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	verbose       bool
	ctx, cancel   = context.WithCancel(context.Background())
	versionsDir   string
	cacheFilePath string
	globalBinDir  string
	Version       = "v0.0.0"
)

func Execute() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Verbose mode enabled")
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println()
		logrus.Debug("Interrupt received, canceling operations...")
		cancel()
		os.Exit(1)
	}()

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

	if err := os.MkdirAll(baseConfigDir, 0755); err != nil {
		logrus.Fatalf("Failed to create config directory: %v", err)
	}
	logrus.Debugf("Config directory ensured: %s", baseConfigDir)

	versionsDir = filepath.Join(baseConfigDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		logrus.Fatalf("Failed to create versions directory: %v", err)
	}
	logrus.Debugf("Versions directory ensured: %s", versionsDir)

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
	if err := os.MkdirAll(baseCacheDir, 0755); err != nil {
		logrus.Fatalf("Failed to create cache directory: %v", err)
	}
	cacheFilePath = filepath.Join(baseCacheDir, "releases.json")
	logrus.Debugf("Cache directory ensured: %s", baseCacheDir)
	logrus.Debugf("Cache file path set: %s", cacheFilePath)

	var baseBinDir string
	if custom := os.Getenv("NVS_BIN_DIR"); custom != "" {
		baseBinDir = custom
		logrus.Debugf("Using custom binary directory from NVS_BIN_DIR: %s", baseBinDir)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Fatalf("Failed to get user home directory: %v", err)
		}
		baseBinDir = filepath.Join(home, ".local", "bin")
		logrus.Debugf("Using default binary directory: %s", baseBinDir)
	}
	if err := os.MkdirAll(baseBinDir, 0755); err != nil {
		logrus.Fatalf("Failed to create binary directory: %v", err)
	}
	globalBinDir = baseBinDir
	logrus.Debugf("Global binary directory ensured: %s", globalBinDir)
}

var rootCmd = &cobra.Command{
	Use:     "nvs",
	Short:   "Neovim version switcher",
	Long:    "A CLI tool to install, switch, list, uninstall, and reset Neovim versions.",
	Version: Version,
}
