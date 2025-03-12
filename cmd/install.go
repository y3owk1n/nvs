package cmd

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/installer"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var installCmd = &cobra.Command{
	Use:     "install <version|stable|nightly>",
	Aliases: []string{"i"},
	Short:   "Install a Neovim version",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Starting installation command")

		alias := releases.NormalizeVersion(args[0])
		logrus.Debugf("Normalized version: %s", alias)
		fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Resolving version %s...", utils.CyanText(alias))))

		if err := installer.InstallVersion(alias, versionsDir, cacheFilePath); err != nil {
			logrus.Fatalf("%v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
