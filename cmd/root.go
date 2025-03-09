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
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println()
		logrus.Info("Interrupt received, canceling operations...")
		cancel()
		os.Exit(1)
	}()

	home, err := os.UserHomeDir()
	if err != nil {
		logrus.Fatalf("Failed to get user home directory: %v", err)
	}
	baseDir := filepath.Join(home, ".nvs")
	versionsDir = filepath.Join(baseDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		logrus.Fatalf("Failed to create versions directory: %v", err)
	}

	globalBinDir = filepath.Join(baseDir, "bin")
	if err := os.MkdirAll(globalBinDir, 0755); err != nil {
		logrus.Fatalf("Failed to create global bin directory: %v", err)
	}

	cacheDir := filepath.Join(baseDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		logrus.Fatalf("Failed to create cache directory: %v", err)
	}
	cacheFilePath = filepath.Join(cacheDir, "releases.json")
}

var rootCmd = &cobra.Command{
	Use:     "nvs",
	Short:   "Neovim version switcher",
	Long:    "A CLI tool to install, switch, list, uninstall, and reset Neovim versions.",
	Version: Version,
}
