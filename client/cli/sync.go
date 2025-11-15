package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/buharamanya/gophkeeper/client/api"
	"github.com/buharamanya/gophkeeper/client/config"
	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/spf13/cobra"
)

var (
	syncForce  bool
	resolveAll bool
	clientWins bool
	serverWins bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize data with server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)
		client.SetToken(cfg.Token)

		// Загружаем локальные данные
		localEntries, lastSync, err := loadLocalData()
		if err != nil {
			fmt.Printf("Error loading local data: %v\n", err)
			return
		}

		if syncForce {
			lastSync = time.Time{} // Принудительная синхронизация всех данных
		}

		fmt.Printf("Starting sync (last sync: %v)\n", lastSync.Format(time.RFC3339))

		// Выполняем синхронизацию
		syncResp, err := client.Sync(lastSync, localEntries)
		if err != nil {
			fmt.Printf("Error during sync: %v\n", err)
			return
		}

		// Обрабатываем конфликты
		if len(syncResp.Conflicts) > 0 {
			fmt.Printf("Found %d conflicts:\n", len(syncResp.Conflicts))
			for i, conflict := range syncResp.Conflicts {
				fmt.Printf("\nConflict %d:\n", i+1)
				fmt.Printf("  Client: %s (v%d, updated: %v)\n",
					conflict.ClientEntry.Name,
					conflict.ClientEntry.Version,
					conflict.ClientEntry.UpdatedAt.Format(time.RFC3339))
				fmt.Printf("  Server: %s (v%d, updated: %v)\n",
					conflict.ServerEntry.Name,
					conflict.ServerEntry.Version,
					conflict.ServerEntry.UpdatedAt.Format(time.RFC3339))

				if resolveAll {
					resolution := "client"
					if serverWins {
						resolution = "server"
					}

					err := client.ResolveConflict(fmt.Sprintf("%d", i), resolution, conflict.ServerEntry)
					if err != nil {
						fmt.Printf("  Error resolving conflict: %v\n", err)
					} else {
						fmt.Printf("  Resolved with %s version\n", resolution)
					}
				}
			}

			if !resolveAll && len(syncResp.Conflicts) > 0 {
				fmt.Println("\nUse --resolve-all with --client-wins or --server-wins to auto-resolve conflicts")
				fmt.Println("Or use 'gophkeeper sync resolve' to resolve conflicts individually")
				return
			}
		}

		// Сохраняем полученные данные
		if err := saveSyncedData(syncResp, syncResp.ServerTime); err != nil {
			fmt.Printf("Error saving synced data: %v\n", err)
			return
		}

		// Показываем результаты
		fmt.Printf("\nSync completed successfully!\n")
		fmt.Printf("Server time: %v\n", syncResp.ServerTime.Format(time.RFC3339))
		if len(syncResp.NewEntries) > 0 {
			fmt.Printf("New entries from server: %d\n", len(syncResp.NewEntries))
		}
		if len(syncResp.UpdatedIDs) > 0 {
			fmt.Printf("Updated entries: %d\n", len(syncResp.UpdatedIDs))
		}
		if len(syncResp.DeletedIDs) > 0 {
			fmt.Printf("Deleted entries: %d\n", len(syncResp.DeletedIDs))
		}
	},
}

var syncStatusCmd = &cobra.Command{
	Use:   "sync-status",
	Short: "Show synchronization status",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)
		client.SetToken(cfg.Token)

		status, err := client.GetSyncStatus()
		if err != nil {
			fmt.Printf("Error getting sync status: %v\n", err)
			return
		}

		fmt.Printf("Synchronization Status:\n")
		fmt.Printf("Last sync: %v\n", status.LastSyncTime.Format(time.RFC3339))
		fmt.Printf("Pending changes: %d\n", status.PendingChanges)
		fmt.Printf("Has conflicts: %t\n", status.HasConflicts)

		if status.PendingChanges > 0 {
			fmt.Println("\nRun 'gophkeeper sync' to synchronize changes")
		}
	},
}

var resolveConflictCmd = &cobra.Command{
	Use:   "sync-resolve [conflict-id]",
	Short: "Resolve synchronization conflict",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		client := api.NewClient(cfg.ServerURL)
		client.SetToken(cfg.Token)

		fmt.Printf("Resolving conflict %s...\n", args[0])
		// TODO: Реализовать интерактивное разрешение конфликта
		fmt.Println("Manual conflict resolution not yet implemented")
	},
}

func init() {
	syncCmd.Flags().BoolVarP(&syncForce, "force", "f", false, "Force full synchronization")
	syncCmd.Flags().BoolVar(&resolveAll, "resolve-all", false, "Automatically resolve all conflicts")
	syncCmd.Flags().BoolVar(&clientWins, "client-wins", false, "Use client version when resolving conflicts")
	syncCmd.Flags().BoolVar(&serverWins, "server-wins", false, "Use server version when resolving conflicts")

	// Убеждаемся, что только один из флагов client-wins/server-wins используется
	syncCmd.MarkFlagsMutuallyExclusive("client-wins", "server-wins")
}

// loadLocalData загружает локальные данные из файла
func loadLocalData() ([]*models.DataEntry, time.Time, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, time.Time{}, err
	}

	appDir := filepath.Join(configDir, "gophkeeper")
	dataFile := filepath.Join(appDir, "local_data.json")
	stateFile := filepath.Join(appDir, "sync_state.json")

	// Создаем директорию если не существует
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return nil, time.Time{}, err
	}

	// Загружаем время последней синхронизации
	var lastSync time.Time
	if stateData, err := os.ReadFile(stateFile); err == nil {
		var state struct {
			LastSyncTime time.Time `json:"last_sync_time"`
		}
		if err := json.Unmarshal(stateData, &state); err == nil {
			lastSync = state.LastSyncTime
		}
	}

	// Загружаем локальные данные
	var entries []*models.DataEntry
	if data, err := os.ReadFile(dataFile); err == nil {
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, lastSync, err
		}
	}

	return entries, lastSync, nil
}

// saveSyncedData сохраняет синхронизированные данные
func saveSyncedData(syncResp *api.SyncResponse, serverTime time.Time) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	appDir := filepath.Join(configDir, "gophkeeper")
	dataFile := filepath.Join(appDir, "local_data.json")
	stateFile := filepath.Join(appDir, "sync_state.json")

	// Сохраняем состояние синхронизации
	state := struct {
		LastSyncTime time.Time `json:"last_sync_time"`
	}{
		LastSyncTime: serverTime,
	}

	stateData, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := os.WriteFile(stateFile, stateData, 0600); err != nil {
		return err
	}

	// TODO: Реализовать логику объединения данных
	// Пока просто сохраняем новые данные от сервера
	if len(syncResp.NewEntries) > 0 {
		data, err := json.Marshal(syncResp.NewEntries)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dataFile, data, 0600); err != nil {
			return err
		}
	}

	return nil
}
