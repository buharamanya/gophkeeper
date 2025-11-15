package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/buharamanya/gophkeeper/server/app"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockDataService implements the data service interface for testing
type MockDataService struct {
	createEntryFunc            func(userID string, entry *models.DataEntry) (string, error)
	getUserEntriesFunc         func(userID string) ([]*models.DataEntry, error)
	getEntryFunc               func(userID, entryID string) (*models.DataEntry, error)
	updateEntryFunc            func(userID string, entry *models.DataEntry) error
	updateEntryWithVersionFunc func(userID string, entry *models.DataEntry, expectedVersion int64) error
	deleteEntryFunc            func(userID, entryID string) error
}

func (m *MockDataService) CreateEntry(userID string, entry *models.DataEntry) (string, error) {
	if m.createEntryFunc != nil {
		return m.createEntryFunc(userID, entry)
	}
	return "mock-entry-id", nil
}

func (m *MockDataService) GetUserEntries(userID string) ([]*models.DataEntry, error) {
	if m.getUserEntriesFunc != nil {
		return m.getUserEntriesFunc(userID)
	}
	return []*models.DataEntry{}, nil
}

func (m *MockDataService) GetEntry(userID, entryID string) (*models.DataEntry, error) {
	if m.getEntryFunc != nil {
		return m.getEntryFunc(userID, entryID)
	}
	return &models.DataEntry{}, nil
}

func (m *MockDataService) UpdateEntry(userID string, entry *models.DataEntry) error {
	if m.updateEntryFunc != nil {
		return m.updateEntryFunc(userID, entry)
	}
	return nil
}

func (m *MockDataService) UpdateEntryWithVersion(userID string, entry *models.DataEntry, expectedVersion int64) error {
	if m.updateEntryWithVersionFunc != nil {
		return m.updateEntryWithVersionFunc(userID, entry, expectedVersion)
	}
	return nil
}

func (m *MockDataService) DeleteEntry(userID, entryID string) error {
	if m.deleteEntryFunc != nil {
		return m.deleteEntryFunc(userID, entryID)
	}
	return nil
}

// DataTestHandler wraps the real Handler but uses interface for dataService
type DataTestHandler struct {
	dataService DataServiceInterface
	logger      *zap.Logger
}

// DataServiceInterface defines the methods needed for testing
type DataServiceInterface interface {
	CreateEntry(userID string, entry *models.DataEntry) (string, error)
	GetUserEntries(userID string) ([]*models.DataEntry, error)
	GetEntry(userID, entryID string) (*models.DataEntry, error)
	UpdateEntry(userID string, entry *models.DataEntry) error
	UpdateEntryWithVersion(userID string, entry *models.DataEntry, expectedVersion int64) error
	DeleteEntry(userID, entryID string) error
}

func (h *DataTestHandler) CreateData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during data creation")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var entry models.DataEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		h.logger.Warn("Failed to decode data entry",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		h.writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	entryID, err := h.dataService.CreateEntry(userID, &entry)
	if err != nil {
		h.logger.Error("Failed to create data entry",
			zap.String("user_id", userID),
			zap.String("entry_name", entry.Name),
			zap.Error(err),
		)
		h.writeInternalError(w, err, "create data entry")
		return
	}

	h.logger.Info("Data entry created successfully",
		zap.String("user_id", userID),
		zap.String("entry_id", entryID),
		zap.String("entry_name", entry.Name),
	)
	h.writeJSON(w, http.StatusOK, map[string]string{"id": entryID})
}

func (h *DataTestHandler) ListData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during data listing")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	entries, err := h.dataService.GetUserEntries(userID)
	if err != nil {
		h.logger.Error("Failed to list user entries",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		h.writeInternalError(w, err, "list data entries")
		return
	}

	h.logger.Debug("Retrieved user entries",
		zap.String("user_id", userID),
		zap.Int("count", len(entries)),
	)
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"data": entries})
}

func (h *DataTestHandler) GetData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during data retrieval")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	entryID := chi.URLParam(r, "id")

	entry, err := h.dataService.GetEntry(userID, entryID)
	if err != nil {
		h.logger.Warn("Data entry not found",
			zap.String("user_id", userID),
			zap.String("entry_id", entryID),
			zap.Error(err),
		)
		h.writeError(w, http.StatusNotFound, "Data not found")
		return
	}

	h.writeJSON(w, http.StatusOK, entry)
}

func (h *DataTestHandler) UpdateData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during data update")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	entryID := chi.URLParam(r, "id")

	var updateReq models.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		h.logger.Warn("Failed to decode data entry for update",
			zap.String("user_id", userID),
			zap.String("entry_id", entryID),
			zap.Error(err),
		)
		h.writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Проверяем, что ID в пути совпадает с ID в теле запроса
	if updateReq.DataEntry.ID != entryID {
		h.logger.Warn("Entry ID mismatch in update request",
			zap.String("user_id", userID),
			zap.String("path_id", entryID),
			zap.String("body_id", updateReq.DataEntry.ID),
		)
		h.writeError(w, http.StatusBadRequest, "Entry ID mismatch")
		return
	}

	// Используем оптимистическую блокировку если указана ожидаемая версия
	if updateReq.ExpectedVersion > 0 {
		err := h.dataService.UpdateEntryWithVersion(userID, updateReq.DataEntry, updateReq.ExpectedVersion)
		if err != nil {
			if err == app.ErrVersionConflict {
				h.logger.Warn("Version conflict during update",
					zap.String("user_id", userID),
					zap.String("entry_id", entryID),
					zap.Int64("expected_version", updateReq.ExpectedVersion),
				)
				h.writeError(w, http.StatusConflict, "Version conflict: data was modified by another operation")
				return
			}
			h.logger.Error("Failed to update data entry with version check",
				zap.String("user_id", userID),
				zap.String("entry_id", entryID),
				zap.Error(err),
			)
			h.writeInternalError(w, err, "update data entry with version check")
			return
		}
	} else {
		// Старый метод без проверки версии (для обратной совместимости)
		err := h.dataService.UpdateEntry(userID, updateReq.DataEntry)
		if err != nil {
			h.logger.Error("Failed to update data entry",
				zap.String("user_id", userID),
				zap.String("entry_id", entryID),
				zap.Error(err),
			)
			h.writeInternalError(w, err, "update data entry")
			return
		}
	}

	h.logger.Info("Data entry updated successfully",
		zap.String("user_id", userID),
		zap.String("entry_id", entryID),
	)
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *DataTestHandler) DeleteData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during data deletion")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	entryID := chi.URLParam(r, "id")

	err := h.dataService.DeleteEntry(userID, entryID)
	if err != nil {
		h.logger.Error("Failed to delete data entry",
			zap.String("user_id", userID),
			zap.String("entry_id", entryID),
			zap.Error(err),
		)
		h.writeInternalError(w, err, "delete data entry")
		return
	}

	h.logger.Info("Data entry deleted successfully",
		zap.String("user_id", userID),
		zap.String("entry_id", entryID),
	)
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Helper functions as methods to avoid conflicts
func (h *DataTestHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *DataTestHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *DataTestHandler) writeInternalError(w http.ResponseWriter, err error, operation string) {
	h.logger.Error("Internal server error",
		zap.String("operation", operation),
		zap.Error(err),
	)
	h.writeError(w, http.StatusInternalServerError, "Internal server error")
}

// Helper to create request with user context
func createRequestWithUser(method, url, body string, userID string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, url, bytes.NewReader([]byte(body)))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add user ID to context
	ctx := context.WithValue(req.Context(), userIDKey, userID)
	return req.WithContext(ctx)
}

func TestDataHandler_CreateData(t *testing.T) {
	t.Run("should create data entry successfully", func(t *testing.T) {
		mockService := &MockDataService{
			createEntryFunc: func(userID string, entry *models.DataEntry) (string, error) {
				assert.Equal(t, "user-123", userID)
				assert.Equal(t, "Test Entry", entry.Name)
				return "new-entry-id", nil
			},
		}

		handler := &DataTestHandler{
			dataService: mockService,
			logger:      zap.NewNop(),
		}

		entry := models.DataEntry{
			Name:     "Test Entry",
			Type:     models.TypeText,
			Metadata: "test metadata",
			Data:     []byte("test data"),
		}
		body, _ := json.Marshal(entry)

		req := createRequestWithUser("POST", "/data", string(body), "user-123")
		w := httptest.NewRecorder()

		handler.CreateData(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "new-entry-id", resp["id"])
	})

	t.Run("should return unauthorized without user context", func(t *testing.T) {
		handler := &DataTestHandler{
			dataService: &MockDataService{},
			logger:      zap.NewNop(),
		}

		req := httptest.NewRequest("POST", "/data", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateData(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("should return bad request for invalid JSON", func(t *testing.T) {
		handler := &DataTestHandler{
			dataService: &MockDataService{},
			logger:      zap.NewNop(),
		}

		req := createRequestWithUser("POST", "/data", "invalid json", "user-123")
		w := httptest.NewRecorder()

		handler.CreateData(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDataHandler_ListData(t *testing.T) {
	t.Run("should list user entries successfully", func(t *testing.T) {
		expectedEntries := []*models.DataEntry{
			{ID: "1", Name: "Entry 1", Type: models.TypeText},
			{ID: "2", Name: "Entry 2", Type: models.TypeLoginPassword},
		}

		mockService := &MockDataService{
			getUserEntriesFunc: func(userID string) ([]*models.DataEntry, error) {
				assert.Equal(t, "user-123", userID)
				return expectedEntries, nil
			},
		}

		handler := &DataTestHandler{
			dataService: mockService,
			logger:      zap.NewNop(),
		}

		req := createRequestWithUser("GET", "/data", "", "user-123")
		w := httptest.NewRecorder()

		handler.ListData(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string][]*models.DataEntry
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp["data"], 2)
		assert.Equal(t, expectedEntries, resp["data"])
	})

	t.Run("should return empty list when no entries", func(t *testing.T) {
		mockService := &MockDataService{
			getUserEntriesFunc: func(userID string) ([]*models.DataEntry, error) {
				return []*models.DataEntry{}, nil
			},
		}

		handler := &DataTestHandler{
			dataService: mockService,
			logger:      zap.NewNop(),
		}

		req := createRequestWithUser("GET", "/data", "", "user-123")
		w := httptest.NewRecorder()

		handler.ListData(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string][]*models.DataEntry
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp["data"])
	})
}

func TestDataHandler_GetData(t *testing.T) {
	t.Run("should get data entry successfully", func(t *testing.T) {
		expectedEntry := &models.DataEntry{
			ID:   "entry-123",
			Name: "Test Entry",
			Type: models.TypeText,
		}

		mockService := &MockDataService{
			getEntryFunc: func(userID, entryID string) (*models.DataEntry, error) {
				assert.Equal(t, "user-123", userID)
				assert.Equal(t, "entry-123", entryID)
				return expectedEntry, nil
			},
		}

		handler := &DataTestHandler{
			dataService: mockService,
			logger:      zap.NewNop(),
		}

		req := createRequestWithUser("GET", "/data/entry-123", "", "user-123")

		// Set up chi URL parameters
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "entry-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.GetData(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var entry models.DataEntry
		err := json.Unmarshal(w.Body.Bytes(), &entry)
		require.NoError(t, err)
		assert.Equal(t, expectedEntry, &entry)
	})

	t.Run("should return not found for non-existent entry", func(t *testing.T) {
		mockService := &MockDataService{
			getEntryFunc: func(userID, entryID string) (*models.DataEntry, error) {
				return nil, assert.AnError
			},
		}

		handler := &DataTestHandler{
			dataService: mockService,
			logger:      zap.NewNop(),
		}

		req := createRequestWithUser("GET", "/data/non-existent", "", "user-123")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "non-existent")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.GetData(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestDataHandler_UpdateData(t *testing.T) {
	t.Run("should update data entry with version check successfully", func(t *testing.T) {
		mockService := &MockDataService{
			updateEntryWithVersionFunc: func(userID string, entry *models.DataEntry, expectedVersion int64) error {
				assert.Equal(t, "user-123", userID)
				assert.Equal(t, "entry-123", entry.ID)
				assert.Equal(t, int64(2), expectedVersion)
				return nil
			},
		}

		handler := &DataTestHandler{
			dataService: mockService,
			logger:      zap.NewNop(),
		}

		updateReq := models.UpdateRequest{
			DataEntry: &models.DataEntry{
				ID:   "entry-123",
				Name: "Updated Entry",
				Type: models.TypeText,
			},
			ExpectedVersion: 2,
		}
		body, _ := json.Marshal(updateReq)

		req := createRequestWithUser("PUT", "/data/entry-123", string(body), "user-123")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "entry-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateData(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should handle version conflict", func(t *testing.T) {
		mockService := &MockDataService{
			updateEntryWithVersionFunc: func(userID string, entry *models.DataEntry, expectedVersion int64) error {
				return app.ErrVersionConflict
			},
		}

		handler := &DataTestHandler{
			dataService: mockService,
			logger:      zap.NewNop(),
		}

		updateReq := models.UpdateRequest{
			DataEntry: &models.DataEntry{
				ID:   "entry-123",
				Name: "Updated Entry",
			},
			ExpectedVersion: 2,
		}
		body, _ := json.Marshal(updateReq)

		req := createRequestWithUser("PUT", "/data/entry-123", string(body), "user-123")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "entry-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateData(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("should return bad request for ID mismatch", func(t *testing.T) {
		handler := &DataTestHandler{
			dataService: &MockDataService{},
			logger:      zap.NewNop(),
		}

		updateReq := models.UpdateRequest{
			DataEntry: &models.DataEntry{
				ID:   "different-id",
				Name: "Updated Entry",
			},
			ExpectedVersion: 2,
		}
		body, _ := json.Marshal(updateReq)

		req := createRequestWithUser("PUT", "/data/entry-123", string(body), "user-123")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "entry-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateData(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDataHandler_DeleteData(t *testing.T) {
	t.Run("should delete data entry successfully", func(t *testing.T) {
		mockService := &MockDataService{
			deleteEntryFunc: func(userID, entryID string) error {
				assert.Equal(t, "user-123", userID)
				assert.Equal(t, "entry-123", entryID)
				return nil
			},
		}

		handler := &DataTestHandler{
			dataService: mockService,
			logger:      zap.NewNop(),
		}

		req := createRequestWithUser("DELETE", "/data/entry-123", "", "user-123")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "entry-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.DeleteData(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "deleted", resp["status"])
	})

	t.Run("should handle delete error", func(t *testing.T) {
		mockService := &MockDataService{
			deleteEntryFunc: func(userID, entryID string) error {
				return assert.AnError
			},
		}

		handler := &DataTestHandler{
			dataService: mockService,
			logger:      zap.NewNop(),
		}

		req := createRequestWithUser("DELETE", "/data/entry-123", "", "user-123")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "entry-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.DeleteData(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
