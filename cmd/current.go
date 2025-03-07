package cmd

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current active version",
	Run: func(cmd *cobra.Command, args []string) {
		current, err := utils.GetCurrentVersion(versionsDir)
		if err != nil {
			logrus.Fatalf("Error getting current version: %v", err)
		}
		fmt.Println(current)
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
