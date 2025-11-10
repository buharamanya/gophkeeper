package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("GophKeeper Client %s\n", Version)
		fmt.Printf("Build date: %s\n", BuildDate)
	},
}
