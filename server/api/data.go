package api

import (
	"encoding/json"
	"net/http"

	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateData(w http.ResponseWriter, r *http.Request) {
	var entry models.DataEntry

	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Используем типизированные ключи контекста
	userID, ok := GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	entryID, err := h.dataService.CreateEntry(userID, &entry)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": entryID})
}

func (h *Handler) ListData(w http.ResponseWriter, r *http.Request) {
	// Используем типизированные ключи контекста
	userID, ok := GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	entries, err := h.dataService.GetUserEntries(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": entries})
}

func (h *Handler) GetData(w http.ResponseWriter, r *http.Request) {
	// Используем типизированные ключи контекста
	userID, ok := GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	entryID := chi.URLParam(r, "id")

	entry, err := h.dataService.GetEntry(userID, entryID)
	if err != nil {
		writeError(w, http.StatusNotFound, "data not found")
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) UpdateData(w http.ResponseWriter, r *http.Request) {
	// Используем типизированные ключи контекста
	userID, ok := GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	entryID := chi.URLParam(r, "id")

	var entry models.DataEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	entry.ID = entryID
	err := h.dataService.UpdateEntry(userID, &entry)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeleteData(w http.ResponseWriter, r *http.Request) {
	// Используем типизированные ключи контекста
	userID, ok := GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	entryID := chi.URLParam(r, "id")

	err := h.dataService.DeleteEntry(userID, entryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
