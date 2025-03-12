package cmd

import (
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print NVS env configurations",
	Long:  "Prints the env configuration used by NVS (NVS_CONFIG_DIR, NVS_CACHE_DIR, and NVS_BIN_DIR).",
	Run: func(cmd *cobra.Command, args []string) {
		configDir := os.Getenv("NVS_CONFIG_DIR")
		if configDir == "" {
			if c, err := os.UserConfigDir(); err == nil {
				configDir = filepath.Join(c, "nvs")
			} else {
				configDir = "Unavailable"
			}
		}

		cacheDir := os.Getenv("NVS_CACHE_DIR")
		if cacheDir == "" {
			if c, err := os.UserCacheDir(); err == nil {
				cacheDir = filepath.Join(c, "nvs")
			} else {
				cacheDir = "Unavailable"
			}
		}

		binDir := os.Getenv("NVS_BIN_DIR")
		if binDir == "" {
			if home, err := os.UserHomeDir(); err == nil {
				binDir = filepath.Join(home, ".local", "bin")
			} else {
				binDir = "Unavailable"
			}
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Variable", "Value"})
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		)
		table.SetTablePadding("1")
		table.SetBorder(false)
		table.SetRowLine(false)
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		table.SetAutoWrapText(false)

		table.Append([]string{
			"NVS_CONFIG_DIR",
			color.New(color.Bold, color.FgCyan).Sprint(configDir),
		})
		table.Append([]string{
			"NVS_CACHE_DIR",
			color.New(color.Bold, color.FgCyan).Sprint(cacheDir),
		})
		table.Append([]string{
			"NVS_BIN_DIR",
			color.New(color.Bold, color.FgCyan).Sprint(binDir),
		})

		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
