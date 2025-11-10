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

	userID := r.Context().Value("user_id").(string)
	entryID, err := h.dataService.CreateEntry(userID, &entry)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": entryID})
}

func (h *Handler) ListData(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	entries, err := h.dataService.GetUserEntries(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": entries})
}

func (h *Handler) GetData(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	entryID := chi.URLParam(r, "id")

	entry, err := h.dataService.GetEntry(userID, entryID)
	if err != nil {
		writeError(w, http.StatusNotFound, "data not found")
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) UpdateData(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
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
	userID := r.Context().Value("user_id").(string)
	entryID := chi.URLParam(r, "id")

	err := h.dataService.DeleteEntry(userID, entryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
