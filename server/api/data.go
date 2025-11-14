package api

import (
	"encoding/json"
	"net/http"

	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func (h *Handler) CreateData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during data creation")
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var entry models.DataEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		h.logger.Warn("Failed to decode data entry",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	entryID, err := h.dataService.CreateEntry(userID, &entry)
	if err != nil {
		h.logger.Error("Failed to create data entry",
			zap.String("user_id", userID),
			zap.String("entry_name", entry.Name),
			zap.Error(err),
		)
		writeInternalError(w, h.logger, err, "create data entry")
		return
	}

	h.logger.Info("Data entry created successfully",
		zap.String("user_id", userID),
		zap.String("entry_id", entryID),
		zap.String("entry_name", entry.Name),
	)
	writeJSON(w, http.StatusOK, map[string]string{"id": entryID})
}

func (h *Handler) ListData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during data listing")
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	entries, err := h.dataService.GetUserEntries(userID)
	if err != nil {
		h.logger.Error("Failed to list user entries",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		writeInternalError(w, h.logger, err, "list data entries")
		return
	}

	h.logger.Debug("Retrieved user entries",
		zap.String("user_id", userID),
		zap.Int("count", len(entries)),
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": entries})
}

func (h *Handler) GetData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during data retrieval")
		writeError(w, http.StatusUnauthorized, "Authentication required")
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
		writeError(w, http.StatusNotFound, "Data not found")
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) UpdateData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during data update")
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	entryID := chi.URLParam(r, "id")

	var entry models.DataEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		h.logger.Warn("Failed to decode data entry for update",
			zap.String("user_id", userID),
			zap.String("entry_id", entryID),
			zap.Error(err),
		)
		writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	entry.ID = entryID
	err := h.dataService.UpdateEntry(userID, &entry)
	if err != nil {
		h.logger.Error("Failed to update data entry",
			zap.String("user_id", userID),
			zap.String("entry_id", entryID),
			zap.Error(err),
		)
		writeInternalError(w, h.logger, err, "update data entry")
		return
	}

	h.logger.Info("Data entry updated successfully",
		zap.String("user_id", userID),
		zap.String("entry_id", entryID),
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeleteData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.logger.Warn("User ID not found in context during data deletion")
		writeError(w, http.StatusUnauthorized, "Authentication required")
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
		writeInternalError(w, h.logger, err, "delete data entry")
		return
	}

	h.logger.Info("Data entry deleted successfully",
		zap.String("user_id", userID),
		zap.String("entry_id", entryID),
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
