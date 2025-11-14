package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/buharamanya/gophkeeper/internal/models"
	"go.uber.org/zap"
)

// SyncRequest содержит данные для синхронизации от клиента
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
	ConflictID  string            `json:"conflict_id"`
}

// SyncStatus представляет статус синхронизации пользователя
type SyncStatus struct {
	LastSyncTime   time.Time `json:"last_sync_time"`
	PendingChanges int       `json:"pending_changes"`
	HasConflicts   bool      `json:"has_conflicts"`
}

func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during sync")
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var syncReq SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&syncReq); err != nil {
		h.logger.Warn("Failed to decode sync request",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Получаем изменения с сервера
	serverChanges, err := h.dataService.GetChangesSince(userID, syncReq.LastSyncTime)
	if err != nil {
		h.logger.Error("Failed to get changes for sync",
			zap.String("user_id", userID),
			zap.Time("since", syncReq.LastSyncTime),
			zap.Error(err),
		)
		writeInternalError(w, h.logger, err, "get sync changes")
		return
	}

	// Проверяем конфликты
	conflicts := h.detectConflicts(syncReq.Entries, serverChanges)

	// Обрабатываем клиентские изменения
	updatedIDs, deletedIDs, err := h.processClientChanges(userID, syncReq.Entries, syncReq.LastSyncTime)
	if err != nil {
		h.logger.Error("Failed to process client changes during sync",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		writeInternalError(w, h.logger, err, "process client sync changes")
		return
	}

	syncResp := SyncResponse{
		ServerTime: time.Now().UTC(),
		Conflicts:  conflicts,
		NewEntries: serverChanges,
		UpdatedIDs: updatedIDs,
		DeletedIDs: deletedIDs,
	}

	h.logger.Info("Sync completed successfully",
		zap.String("user_id", userID),
		zap.Int("new_entries", len(serverChanges)),
		zap.Int("conflicts", len(conflicts)),
	)
	writeJSON(w, http.StatusOK, syncResp)
}

func (h *Handler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during sync status check")
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Получаем время последней синхронизации
	lastSync, pendingChanges, hasConflicts, err := h.dataService.GetSyncStatus(userID)
	if err != nil {
		h.logger.Error("Failed to get sync status",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		writeInternalError(w, h.logger, err, "get sync status")
		return
	}

	status := SyncStatus{
		LastSyncTime:   lastSync,
		PendingChanges: pendingChanges,
		HasConflicts:   hasConflicts,
	}

	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) ResolveConflict(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during conflict resolution")
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var resolutionReq struct {
		ConflictID string            `json:"conflict_id"`
		Resolution string            `json:"resolution"` // "client", "server"
		Entry      *models.DataEntry `json:"entry"`
	}

	if err := json.NewDecoder(r.Body).Decode(&resolutionReq); err != nil {
		h.logger.Warn("Failed to decode conflict resolution request",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Обрабатываем разрешение конфликта
	err := h.dataService.ResolveConflict(userID, resolutionReq.ConflictID, resolutionReq.Resolution, resolutionReq.Entry)
	if err != nil {
		h.logger.Error("Failed to resolve conflict",
			zap.String("user_id", userID),
			zap.String("conflict_id", resolutionReq.ConflictID),
			zap.Error(err),
		)
		writeInternalError(w, h.logger, err, "resolve conflict")
		return
	}

	h.logger.Info("Conflict resolved successfully",
		zap.String("user_id", userID),
		zap.String("conflict_id", resolutionReq.ConflictID),
		zap.String("resolution", resolutionReq.Resolution),
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// detectConflicts обнаруживает конфликты между клиентскими и серверными данными
func (h *Handler) detectConflicts(clientEntries, serverEntries []*models.DataEntry) []*Conflict {
	var conflicts []*Conflict

	// Создаем карту серверных записей для быстрого поиска
	serverMap := make(map[string]*models.DataEntry)
	for _, entry := range serverEntries {
		serverMap[entry.ID] = entry
	}

	// Проверяем каждую клиентскую запись на конфликты
	for _, clientEntry := range clientEntries {
		if serverEntry, exists := serverMap[clientEntry.ID]; exists {
			// Если версии разные и обе записи были изменены после последней синхронизации
			if clientEntry.Version != serverEntry.Version &&
				clientEntry.UpdatedAt.After(clientEntry.LastSyncTime) &&
				serverEntry.UpdatedAt.After(clientEntry.LastSyncTime) {
				conflicts = append(conflicts, &Conflict{
					ClientEntry: clientEntry,
					ServerEntry: serverEntry,
					ConflictID:  clientEntry.ID + "_" + clientEntry.UpdatedAt.Format(time.RFC3339),
				})
			}
		}
	}

	return conflicts
}

// processClientChanges обрабатывает изменения от клиента
func (h *Handler) processClientChanges(userID string, clientEntries []*models.DataEntry, lastSyncTime time.Time) ([]string, []string, error) {
	var updatedIDs, deletedIDs []string

	for _, entry := range clientEntries {
		// Пропускаем записи, которые не были изменены после последней синхронизации
		if !entry.UpdatedAt.After(lastSyncTime) {
			continue
		}

		if entry.IsDeleted {
			// Удаляем запись
			if err := h.dataService.DeleteEntry(userID, entry.ID); err == nil {
				deletedIDs = append(deletedIDs, entry.ID)
			}
		} else if entry.LastSyncTime.IsZero() {
			// Новая запись
			if _, err := h.dataService.CreateEntry(userID, entry); err == nil {
				updatedIDs = append(updatedIDs, entry.ID)
			}
		} else {
			// Обновление существующей записи
			if err := h.dataService.UpdateEntry(userID, entry); err == nil {
				updatedIDs = append(updatedIDs, entry.ID)
			}
		}
	}

	return updatedIDs, deletedIDs, nil
}
