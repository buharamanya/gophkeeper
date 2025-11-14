package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/buharamanya/gophkeeper/client/api"
	"github.com/buharamanya/gophkeeper/client/config"
	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/spf13/cobra"
)

var (
	dataType        string
	name            string
	metadata        string
	inputFile       string
	expectedVersion int64 // Новый флаг для ожидаемой версии
)

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Store new data entry",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)
		client.SetToken(cfg.Token)

		var data []byte
		var err error

		if inputFile != "" {
			data, err = os.ReadFile(inputFile)
			if err != nil {
				fmt.Printf("Error reading file: %v\n", err)
				return
			}
		} else {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Printf("Error reading from stdin: %v\n", err)
				return
			}
		}

		entry := &models.DataEntry{
			Name:     name,
			Type:     models.DataType(dataType),
			Metadata: metadata,
			Data:     data,
			Version:  1, // Начальная версия всегда 1
		}

		id, err := client.CreateData(entry)
		if err != nil {
			fmt.Printf("Error storing data: %v\n", err)
			return
		}

		fmt.Printf("Data stored successfully. ID: %s, Version: 1\n", id)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all data entries",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)
		client.SetToken(cfg.Token)

		entries, err := client.ListData()
		if err != nil {
			fmt.Printf("Error listing data: %v\n", err)
			return
		}

		if len(entries) == 0 {
			fmt.Println("No data entries found")
			return
		}

		for _, entry := range entries {
			fmt.Printf("ID: %s, Name: %s, Type: %s, Version: %d, Created: %s\n",
				entry.ID, entry.Name, entry.Type, entry.Version, entry.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	},
}

var getCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get data entry by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)
		client.SetToken(cfg.Token)

		entry, err := client.GetData(args[0])
		if err != nil {
			fmt.Printf("Error getting data: %v\n", err)
			return
		}

		fmt.Printf("ID: %s\n", entry.ID)
		fmt.Printf("Name: %s\n", entry.Name)
		fmt.Printf("Type: %s\n", entry.Type)
		fmt.Printf("Version: %d\n", entry.Version)
		fmt.Printf("Created: %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated: %s\n", entry.UpdatedAt.Format("2006-01-02 15:04:05"))

		if entry.Metadata != "" {
			fmt.Printf("Metadata: %s\n", entry.Metadata)
		}

		// Для текстовых данных показываем содержимое
		if entry.Type == models.TypeText {
			fmt.Printf("Data: %s\n", string(entry.Data))
		} else {
			fmt.Printf("Data size: %d bytes\n", len(entry.Data))
		}
	},
}

var updateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update data entry with optimistic locking",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)
		client.SetToken(cfg.Token)

		var data []byte
		var err error

		if inputFile != "" {
			data, err = os.ReadFile(inputFile)
			if err != nil {
				fmt.Printf("Error reading file: %v\n", err)
				return
			}
		} else {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Printf("Error reading from stdin: %v\n", err)
				return
			}
		}

		// Сначала получаем текущую запись чтобы узнать версию
		currentEntry, err := client.GetData(args[0])
		if err != nil {
			fmt.Printf("Error getting current data: %v\n", err)
			return
		}

		entry := &models.DataEntry{
			ID:       args[0],
			Name:     name,
			Type:     models.DataType(dataType),
			Metadata: metadata,
			Data:     data,
			Version:  currentEntry.Version, // Используем текущую версию
		}

		// Если не указана ожидаемая версия, используем текущую
		if expectedVersion == 0 {
			expectedVersion = currentEntry.Version
		}

		updateReq := &models.UpdateRequest{
			DataEntry:       entry,
			ExpectedVersion: expectedVersion,
		}

		// Используем новый метод с поддержкой оптимистической блокировки
		err = client.UpdateDataWithVersion(args[0], updateReq)
		if err != nil {
			fmt.Printf("Error updating data: %v\n", err)
			return
		}

		fmt.Printf("Data updated successfully. New version: %d\n", currentEntry.Version+1)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete data entry",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)
		client.SetToken(cfg.Token)

		err := client.DeleteData(args[0])
		if err != nil {
			fmt.Printf("Error deleting data: %v\n", err)
			return
		}

		fmt.Println("Data deleted successfully")
	},
}

func init() {
	storeCmd.Flags().StringVarP(&dataType, "type", "t", "text", "Data type (login_password, text, binary, card)")
	storeCmd.Flags().StringVarP(&name, "name", "n", "", "Entry name")
	storeCmd.Flags().StringVarP(&metadata, "metadata", "m", "", "Metadata")
	storeCmd.Flags().StringVarP(&inputFile, "file", "f", "", "Input file (default: stdin)")
	storeCmd.MarkFlagRequired("name")

	updateCmd.Flags().StringVarP(&dataType, "type", "t", "text", "Data type (login_password, text, binary, card)")
	updateCmd.Flags().StringVarP(&name, "name", "n", "", "Entry name")
	updateCmd.Flags().StringVarP(&metadata, "metadata", "m", "", "Metadata")
	updateCmd.Flags().StringVarP(&inputFile, "file", "f", "", "Input file (default: stdin)")
	updateCmd.Flags().Int64VarP(&expectedVersion, "expected-version", "v", 0, "Expected version for optimistic locking (default: current version)")
}
