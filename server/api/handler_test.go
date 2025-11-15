package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buharamanya/gophkeeper/server/app"
	"github.com/buharamanya/gophkeeper/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewHandler(t *testing.T) {
	t.Run("should create handler with all dependencies", func(t *testing.T) {
		cfg := &config.Config{}
		logger := zap.NewNop()
		authService := &app.AuthService{}
		dataService := &app.DataService{}

		handler := NewHandler(authService, dataService, cfg, logger)

		assert.NotNil(t, handler)
		assert.Equal(t, authService, handler.authService)
		assert.Equal(t, dataService, handler.dataService)
		assert.Equal(t, cfg, handler.cfg)
		assert.Equal(t, logger, handler.logger)
	})
}

func TestHandler_HealthCheck(t *testing.T) {
	t.Run("should return ok status", func(t *testing.T) {
		handler := &Handler{
			logger: zap.NewNop(),
		}

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler.healthCheck(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
	})
}

func TestHandler_NotFoundHandler(t *testing.T) {
	t.Run("should return 404 for unknown routes", func(t *testing.T) {
		handler := &Handler{
			logger: zap.NewNop(),
		}

		req := httptest.NewRequest("GET", "/unknown-route", nil)
		w := httptest.NewRecorder()

		handler.notFoundHandler(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Resource not found", response.Error)
	})
}

func TestHandler_MethodNotAllowedHandler(t *testing.T) {
	t.Run("should return 405 for unsupported methods", func(t *testing.T) {
		handler := &Handler{
			logger: zap.NewNop(),
		}

		req := httptest.NewRequest("PATCH", "/api/data", nil)
		w := httptest.NewRecorder()

		handler.methodNotAllowedHandler(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Method not allowed", response.Error)
	})
}

func TestWriteJSON(t *testing.T) {
	t.Run("should write JSON response with correct headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"message": "test"}

		writeJSON(w, http.StatusOK, data)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test", response["message"])
	})
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		message  string
		expected string
	}{
		{
			name:     "should write client error with custom message",
			status:   http.StatusBadRequest,
			message:  "Invalid input",
			expected: "Invalid input",
		},
		{
			name:     "should write server error with generic message",
			status:   http.StatusInternalServerError,
			message:  "Database error",
			expected: "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			writeError(w, tt.status, tt.message)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, response.Error)
		})
	}
}

func TestHandler_RecoveryMiddleware(t *testing.T) {
	t.Run("should recover from panic and return 500", func(t *testing.T) {
		handler := &Handler{
			logger: zap.NewNop(),
		}

		// Handler that panics
		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		// Wrap with recovery middleware
		recoveryHandler := handler.recoveryMiddleware(panicHandler)
		recoveryHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		// Исправлено: ожидаем стандартное сообщение HTTP для 500 ошибки
		assert.Equal(t, "Internal Server Error", response.Error)
	})

	t.Run("should pass through without panic", func(t *testing.T) {
		handler := &Handler{
			logger: zap.NewNop(),
		}

		normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		recoveryHandler := handler.recoveryMiddleware(normalHandler)
		recoveryHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_Shutdown(t *testing.T) {
	t.Run("should handle nil server gracefully", func(t *testing.T) {
		handler := &Handler{
			logger: zap.NewNop(),
		}

		ctx := context.Background()
		err := handler.Shutdown(ctx)

		assert.NoError(t, err)
	})
}

func TestErrorResponse(t *testing.T) {
	t.Run("should marshal error response correctly", func(t *testing.T) {
		errorResp := ErrorResponse{
			Error: "test error",
		}

		data, err := json.Marshal(errorResp)
		require.NoError(t, err)

		var unmarshaled ErrorResponse
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, "test error", unmarshaled.Error)
	})
}
