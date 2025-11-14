package api

import (
	"encoding/json"
	"net/http"

	"github.com/buharamanya/gophkeeper/internal/models"
	"go.uber.org/zap"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Registration request", zap.String("remote_addr", r.RemoteAddr))

	var req models.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode registration request", zap.Error(err))
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	resp, err := h.authService.Register(req.Login, req.Password)
	if err != nil {
		h.logger.Warn("Registration failed", zap.String("login", req.Login), zap.Error(err))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("User registered successfully",
		zap.String("user_id", resp.UserID),
		zap.String("login", req.Login),
	)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Login request", zap.String("remote_addr", r.RemoteAddr))

	var req models.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode login request", zap.Error(err))
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	resp, err := h.authService.Login(req.Login, req.Password)
	if err != nil {
		h.logger.Warn("Login failed", zap.String("login", req.Login), zap.Error(err))
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.logger.Info("User logged in successfully",
		zap.String("user_id", resp.UserID),
		zap.String("login", req.Login),
	)
	writeJSON(w, http.StatusOK, resp)
}
