package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset all data (remove symlinks, downloaded versions, cache, etc.)",
	Long:  "WARNING: This command will delete the entire ~/.nvs directory and all its contents. Use with caution.",
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Fatalf("Failed to get home directory: %v", err)
		}
		baseDir := filepath.Join(home, ".nvs")
		fmt.Printf("WARNING: This will delete all data in %s. Are you sure? (y/N): ", baseDir)
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(answer) != "y" {
			fmt.Println("Reset cancelled.")
			return
		}
		if err := os.RemoveAll(baseDir); err != nil {
			logrus.Fatalf("Failed to reset data: %v", err)
		}
		logrus.Info("Reset successful. All data has been cleared.")
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
