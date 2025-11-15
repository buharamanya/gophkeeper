package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// SyncDataServiceInterface для тестов синхронизации (уникальное имя)
type SyncDataServiceInterface interface {
	GetChangesSince(userID string, since time.Time) ([]*models.DataEntry, error)
	GetSyncStatus(userID string) (time.Time, int, bool, error)
	ResolveConflict(userID, conflictID, resolution string, entry *models.DataEntry) error
	DeleteEntry(userID, entryID string) error
	CreateEntry(userID string, entry *models.DataEntry) (*models.DataEntry, error)
	UpdateEntry(userID string, entry *models.DataEntry) error
}

// TestSyncHandler для тестов синхронизации
type TestSyncHandler struct {
	dataService SyncDataServiceInterface
	logger      *zap.Logger
}

// SyncMockDataService для тестов синхронизации
type SyncMockDataService struct {
	GetChangesSinceFunc func(userID string, since time.Time) ([]*models.DataEntry, error)
	GetSyncStatusFunc   func(userID string) (time.Time, int, bool, error)
	ResolveConflictFunc func(userID, conflictID, resolution string, entry *models.DataEntry) error
	DeleteEntryFunc     func(userID, entryID string) error
	CreateEntryFunc     func(userID string, entry *models.DataEntry) (*models.DataEntry, error)
	UpdateEntryFunc     func(userID string, entry *models.DataEntry) error
}

func (m *SyncMockDataService) GetChangesSince(userID string, since time.Time) ([]*models.DataEntry, error) {
	if m.GetChangesSinceFunc != nil {
		return m.GetChangesSinceFunc(userID, since)
	}
	return []*models.DataEntry{}, nil
}

func (m *SyncMockDataService) GetSyncStatus(userID string) (time.Time, int, bool, error) {
	if m.GetSyncStatusFunc != nil {
		return m.GetSyncStatusFunc(userID)
	}
	return time.Time{}, 0, false, nil
}

func (m *SyncMockDataService) ResolveConflict(userID, conflictID, resolution string, entry *models.DataEntry) error {
	if m.ResolveConflictFunc != nil {
		return m.ResolveConflictFunc(userID, conflictID, resolution, entry)
	}
	return nil
}

func (m *SyncMockDataService) DeleteEntry(userID, entryID string) error {
	if m.DeleteEntryFunc != nil {
		return m.DeleteEntryFunc(userID, entryID)
	}
	return nil
}

func (m *SyncMockDataService) CreateEntry(userID string, entry *models.DataEntry) (*models.DataEntry, error) {
	if m.CreateEntryFunc != nil {
		return m.CreateEntryFunc(userID, entry)
	}
	return entry, nil
}

func (m *SyncMockDataService) UpdateEntry(userID string, entry *models.DataEntry) error {
	if m.UpdateEntryFunc != nil {
		return m.UpdateEntryFunc(userID, entry)
	}
	return nil
}

// Копируем методы из оригинального Handler для тестов
func (h *TestSyncHandler) Sync(w http.ResponseWriter, r *http.Request) {
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

func (h *TestSyncHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
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

func (h *TestSyncHandler) ResolveConflict(w http.ResponseWriter, r *http.Request) {
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
func (h *TestSyncHandler) detectConflicts(clientEntries, serverEntries []*models.DataEntry) []*Conflict {
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
func (h *TestSyncHandler) processClientChanges(userID string, clientEntries []*models.DataEntry, lastSyncTime time.Time) ([]string, []string, error) {
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

// Тесты
func TestSync(t *testing.T) {
	logger := zap.NewNop()
	now := time.Now().UTC()

	tests := []struct {
		name           string
		setupMock      func() *SyncMockDataService
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, resp *http.Response)
	}{
		{
			name: "successful sync",
			setupMock: func() *SyncMockDataService {
				return &SyncMockDataService{
					GetChangesSinceFunc: func(userID string, since time.Time) ([]*models.DataEntry, error) {
						return []*models.DataEntry{
							{
								ID:        "server-entry-1",
								Version:   1,
								UpdatedAt: now,
							},
						}, nil
					},
				}
			},
			requestBody: SyncRequest{
				LastSyncTime: now.Add(-time.Hour),
				Entries: []*models.DataEntry{
					{
						ID:           "client-entry-1",
						Version:      1,
						UpdatedAt:    now,
						LastSyncTime: now.Add(-time.Hour),
					},
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var syncResp SyncResponse
				err := json.NewDecoder(resp.Body).Decode(&syncResp)
				require.NoError(t, err)
				assert.Len(t, syncResp.NewEntries, 1)
				assert.Len(t, syncResp.Conflicts, 0)
			},
		},
		{
			name: "sync with conflicts",
			setupMock: func() *SyncMockDataService {
				return &SyncMockDataService{
					GetChangesSinceFunc: func(userID string, since time.Time) ([]*models.DataEntry, error) {
						return []*models.DataEntry{
							{
								ID:        "conflict-entry",
								Version:   2, // Другая версия
								UpdatedAt: now,
							},
						}, nil
					},
				}
			},
			requestBody: SyncRequest{
				LastSyncTime: now.Add(-time.Hour),
				Entries: []*models.DataEntry{
					{
						ID:           "conflict-entry",
						Version:      1, // Другая версия
						UpdatedAt:    now,
						LastSyncTime: now.Add(-time.Hour),
					},
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var syncResp SyncResponse
				err := json.NewDecoder(resp.Body).Decode(&syncResp)
				require.NoError(t, err)
				assert.Len(t, syncResp.Conflicts, 1)
			},
		},
		{
			name: "invalid request body",
			setupMock: func() *SyncMockDataService {
				return &SyncMockDataService{}
			},
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error during sync",
			setupMock: func() *SyncMockDataService {
				return &SyncMockDataService{
					GetChangesSinceFunc: func(userID string, since time.Time) ([]*models.DataEntry, error) {
						return nil, assert.AnError
					},
				}
			},
			requestBody: SyncRequest{
				LastSyncTime: now.Add(-time.Hour),
				Entries:      []*models.DataEntry{},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDataService := tt.setupMock()

			handler := &TestSyncHandler{
				logger:      logger,
				dataService: mockDataService,
			}

			var bodyBytes []byte
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest("POST", "/sync", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Добавляем userID в контекст
			ctx := WithUserID(req.Context(), "test-user")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.Sync(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Result())
			}
		})
	}
}

func TestGetSyncStatus(t *testing.T) {
	logger := zap.NewNop()
	lastSync := time.Now().UTC().Add(-time.Hour)

	tests := []struct {
		name           string
		setupMock      func() *SyncMockDataService
		expectedStatus int
		checkResponse  func(t *testing.T, resp *http.Response)
	}{
		{
			name: "successful status check",
			setupMock: func() *SyncMockDataService {
				return &SyncMockDataService{
					GetSyncStatusFunc: func(userID string) (time.Time, int, bool, error) {
						return lastSync, 5, true, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var status SyncStatus
				err := json.NewDecoder(resp.Body).Decode(&status)
				require.NoError(t, err)
				assert.Equal(t, lastSync, status.LastSyncTime)
				assert.Equal(t, 5, status.PendingChanges)
				assert.True(t, status.HasConflicts)
			},
		},
		{
			name: "service error during status check",
			setupMock: func() *SyncMockDataService {
				return &SyncMockDataService{
					GetSyncStatusFunc: func(userID string) (time.Time, int, bool, error) {
						return time.Time{}, 0, false, assert.AnError
					},
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDataService := tt.setupMock()

			handler := &TestSyncHandler{
				logger:      logger,
				dataService: mockDataService,
			}

			req := httptest.NewRequest("GET", "/sync/status", nil)

			// Добавляем userID в контекст
			ctx := WithUserID(req.Context(), "test-user")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.GetSyncStatus(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Result())
			}
		})
	}
}

func TestResolveConflict(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		setupMock      func() *SyncMockDataService
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "successful conflict resolution",
			setupMock: func() *SyncMockDataService {
				return &SyncMockDataService{
					ResolveConflictFunc: func(userID, conflictID, resolution string, entry *models.DataEntry) error {
						return nil
					},
				}
			},
			requestBody: map[string]interface{}{
				"conflict_id": "conflict-123",
				"resolution":  "client",
				"entry": map[string]interface{}{
					"id":      "entry-1",
					"version": 1,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid request body",
			setupMock: func() *SyncMockDataService {
				return &SyncMockDataService{}
			},
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error during conflict resolution",
			setupMock: func() *SyncMockDataService {
				return &SyncMockDataService{
					ResolveConflictFunc: func(userID, conflictID, resolution string, entry *models.DataEntry) error {
						return assert.AnError
					},
				}
			},
			requestBody: map[string]interface{}{
				"conflict_id": "conflict-123",
				"resolution":  "client",
				"entry":       map[string]interface{}{},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDataService := tt.setupMock()

			handler := &TestSyncHandler{
				logger:      logger,
				dataService: mockDataService,
			}

			var bodyBytes []byte
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest("POST", "/sync/resolve", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Добавляем userID в контекст
			ctx := WithUserID(req.Context(), "test-user")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.ResolveConflict(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestSync_Unauthorized(t *testing.T) {
	logger := zap.NewNop()
	handler := &TestSyncHandler{
		logger: logger,
	}

	// Тест без userID в контексте
	req := httptest.NewRequest("POST", "/sync", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handler.Sync(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDetectConflicts(t *testing.T) {
	handler := &TestSyncHandler{logger: zap.NewNop()}
	now := time.Now().UTC()
	lastSync := now.Add(-time.Hour)

	tests := []struct {
		name          string
		clientEntries []*models.DataEntry
		serverEntries []*models.DataEntry
		expectedCount int
	}{
		{
			name: "no conflicts - different entries",
			clientEntries: []*models.DataEntry{
				{ID: "entry-1", Version: 1, UpdatedAt: now, LastSyncTime: lastSync},
			},
			serverEntries: []*models.DataEntry{
				{ID: "entry-2", Version: 1, UpdatedAt: now, LastSyncTime: lastSync},
			},
			expectedCount: 0,
		},
		{
			name: "conflict detected",
			clientEntries: []*models.DataEntry{
				{ID: "entry-1", Version: 1, UpdatedAt: now, LastSyncTime: lastSync},
			},
			serverEntries: []*models.DataEntry{
				{ID: "entry-1", Version: 2, UpdatedAt: now, LastSyncTime: lastSync},
			},
			expectedCount: 1,
		},
		{
			name: "no conflict - same version",
			clientEntries: []*models.DataEntry{
				{ID: "entry-1", Version: 1, UpdatedAt: now, LastSyncTime: lastSync},
			},
			serverEntries: []*models.DataEntry{
				{ID: "entry-1", Version: 1, UpdatedAt: now, LastSyncTime: lastSync},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts := handler.detectConflicts(tt.clientEntries, tt.serverEntries)
			assert.Len(t, conflicts, tt.expectedCount)
		})
	}
}
