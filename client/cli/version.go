package cli

import (
	"fmt"

	"github.com/buharamanya/gophkeeper/client/config"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and configuration information",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()

		fmt.Printf("GophKeeper Client %s\n", Version)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("Server URL: %s\n", cfg.ServerURL)

		configPath, err := config.GetConfigPath()
		if err == nil {
			fmt.Printf("Config file: %s\n", configPath)
		}

		if cfg.Token != "" {
			fmt.Printf("Status: Authenticated ✓\n")
		} else {
			fmt.Printf("Status: Not authenticated\n")
		}
	},
}
