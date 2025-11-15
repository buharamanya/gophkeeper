package cli

import (
	"github.com/spf13/cobra"
)

var (
	Version   string
	BuildDate string
)

// RootCmd - корневая команда, экспортируемая для использования в других файлах пакета
var RootCmd = &cobra.Command{
	Use:   "gophkeeper",
	Short: "Secure password manager",
}

func Execute(version, buildDate string) error {
	Version = version
	BuildDate = buildDate

	// Добавляем все команды к корневой команде
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(registerCmd)
	RootCmd.AddCommand(loginCmd)
	RootCmd.AddCommand(logoutCmd)
	RootCmd.AddCommand(statusCmd)
	RootCmd.AddCommand(storeCmd)
	RootCmd.AddCommand(listCmd)
	RootCmd.AddCommand(getCmd)
	RootCmd.AddCommand(updateCmd)
	RootCmd.AddCommand(deleteCmd)
	RootCmd.AddCommand(syncCmd)
	RootCmd.AddCommand(syncStatusCmd)
	RootCmd.AddCommand(resolveConflictCmd)

	return RootCmd.Execute()
}
