package api

import (
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.logger.Warn("Authorization header missing", zap.String("path", r.URL.Path))
			writeError(w, http.StatusUnauthorized, "authorization header required")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			h.logger.Warn("Bearer token missing", zap.String("path", r.URL.Path))
			writeError(w, http.StatusUnauthorized, "bearer token required")
			return
		}

		claims, err := h.authService.ValidateToken(tokenString)
		if err != nil {
			h.logger.Warn("Invalid token", zap.String("path", r.URL.Path), zap.Error(err))
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		// Логируем успешную аутентификацию
		h.logger.Debug("User authenticated",
			zap.String("user_id", claims.UserID),
			zap.String("login", claims.Login),
			zap.String("path", r.URL.Path),
		)

		// Используем типизированные ключи контекста
		ctx := WithUserID(r.Context(), claims.UserID)
		ctx = WithUserLogin(ctx, claims.Login)

		// Логируем время выполнения
		duration := time.Since(start)
		h.logger.Debug("Auth middleware completed",
			zap.String("user_id", claims.UserID),
			zap.Duration("duration", duration),
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
