package cli

import (
	"github.com/spf13/cobra"
)

var (
	Version   string
	BuildDate string
)

func Execute(version, buildDate string) error {
	Version = version
	BuildDate = buildDate

	rootCmd := &cobra.Command{
		Use:   "gophkeeper",
		Short: "Secure password manager",
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(storeCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(deleteCmd)

	return rootCmd.Execute()
}
