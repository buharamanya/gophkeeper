package api

import (
	"net/http"
	"strings"
)

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "authorization header required")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			writeError(w, http.StatusUnauthorized, "bearer token required")
			return
		}

		claims, err := h.authService.ValidateToken(tokenString)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		// Используем типизированные ключи контекста
		ctx := WithUserID(r.Context(), claims.UserID)
		ctx = WithUserLogin(ctx, claims.Login)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
