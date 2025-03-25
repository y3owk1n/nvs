package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/builder"
	"github.com/y3owk1n/nvs/pkg/installer"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var useCmd = &cobra.Command{
	Use:   "use <version|stable|nightly|commit-hash>",
	Short: "Switch to a specific version or commit hash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Starting use command")

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
		defer cancel()

		alias := releases.NormalizeVersion(args[0])
		targetVersion := alias

		logrus.Debugf("Resolved target version: %s", targetVersion)

		if !utils.IsInstalled(versionsDir, targetVersion) {
			isCommitHash := releases.IsCommitHash(alias)
			logrus.Debugf("isCommitHash: %t", isCommitHash)

			if isCommitHash {
				logrus.Debugf("Building Neovim from commit %s", alias)
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("Building Neovim from commit "+utils.CyanText(alias)))
				if err := builder.BuildFromCommit(ctx, alias, versionsDir); err != nil {
					logrus.Fatalf("%v", err)
				}
			} else {
				logrus.Debugf("Start installing %s", alias)
				if err := installer.InstallVersion(ctx, alias, versionsDir, cacheFilePath); err != nil {
					logrus.Fatalf("%v", err)
				}
			}
		}

		currentSymlink := filepath.Join(versionsDir, "current")
		if current, err := os.Readlink(currentSymlink); err == nil {
			if filepath.Base(current) == targetVersion {
				fmt.Printf("%s Already using Neovim %s\n", utils.WarningIcon(), utils.CyanText(targetVersion))
				logrus.Debugf("Already using version: %s", targetVersion)
				return
			}
		}

		if err := utils.UseVersion(targetVersion, currentSymlink, versionsDir, globalBinDir); err != nil {
			logrus.Fatalf("%v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
