package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/buharamanya/gophkeeper/internal/models"
)

// SyncRequest содержит данные для синхронизации
type SyncRequest struct {
	LastSyncTime time.Time           `json:"last_sync_time"`
	Entries      []*models.DataEntry `json:"entries"`
}

// SyncResponse содержит результаты синхронизации
type SyncResponse struct {
	ServerTime time.Time           `json:"server_time"`
	Conflicts  []*Conflict         `json:"conflicts,omitempty"`
	NewEntries []*models.DataEntry `json:"new_entries,omitempty"`
	UpdatedIDs []string            `json:"updated_ids,omitempty"`
	DeletedIDs []string            `json:"deleted_ids,omitempty"`
}

// Conflict представляет конфликт синхронизации
type Conflict struct {
	ClientEntry *models.DataEntry `json:"client_entry"`
	ServerEntry *models.DataEntry `json:"server_entry"`
	Resolution  string            `json:"resolution,omitempty"` // "client", "server", "manual"
}

// SyncStatus представляет статус синхронизации
type SyncStatus struct {
	LastSyncTime   time.Time `json:"last_sync_time"`
	PendingChanges int       `json:"pending_changes"`
	HasConflicts   bool      `json:"has_conflicts"`
}

// Sync выполняет синхронизацию данных с сервером
func (c *Client) Sync(lastSyncTime time.Time, localEntries []*models.DataEntry) (*SyncResponse, error) {
	req := SyncRequest{
		LastSyncTime: lastSyncTime,
		Entries:      localEntries,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sync request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, "/api/sync", body)
	if err != nil {
		return nil, fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sync failed with status: %d", resp.StatusCode)
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return nil, fmt.Errorf("failed to decode sync response: %w", err)
	}

	return &syncResp, nil
}

// GetSyncStatus получает статус синхронизации
func (c *Client) GetSyncStatus() (*SyncStatus, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/sync/status", nil)
	if err != nil {
		return nil, fmt.Errorf("sync status request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sync status failed with status: %d", resp.StatusCode)
	}

	var status SyncStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode sync status: %w", err)
	}

	return &status, nil
}

// ResolveConflict разрешает конфликт синхронизации
func (c *Client) ResolveConflict(conflictID string, resolution string, entry *models.DataEntry) error {
	resolutionReq := map[string]interface{}{
		"conflict_id": conflictID,
		"resolution":  resolution,
		"entry":       entry,
	}

	body, err := json.Marshal(resolutionReq)
	if err != nil {
		return fmt.Errorf("failed to marshal conflict resolution: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, "/api/sync/resolve", body)
	if err != nil {
		return fmt.Errorf("conflict resolution failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("conflict resolution failed with status: %d", resp.StatusCode)
	}

	return nil
}
