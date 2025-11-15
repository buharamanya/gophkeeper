package api

import (
	"net/http"

	"go.uber.org/zap"
)

// GracefulMiddleware проверяет контекст запроса на предмет отмены
func (h *Handler) GracefulMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем не был ли отменен контекст запроса
		select {
		case <-r.Context().Done():
			h.logger.Warn("Request context cancelled before processing",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)
			writeError(w, http.StatusServiceUnavailable, "Service is shutting down")
			return
		default:
			// Продолжаем обработку
			next.ServeHTTP(w, r)
		}
	})
}
