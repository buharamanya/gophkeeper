package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/buharamanya/gophkeeper/internal/models"
	"github.com/buharamanya/gophkeeper/server/app"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// AuthServiceInterface defines the methods needed for testing
type AuthServiceInterface interface {
	Register(login, password string) (*models.AuthResponse, error)
	Login(login, password string) (*models.AuthResponse, error)
	ValidateToken(tokenString string) (*app.Claims, error)
}

// TestHandler wraps the real Handler but uses interface for authService
type TestHandler struct {
	authService AuthServiceInterface
	logger      *zap.Logger
}

// MockAuthService implements AuthServiceInterface
type MockAuthService struct {
	registerFunc      func(login, password string) (*models.AuthResponse, error)
	loginFunc         func(login, password string) (*models.AuthResponse, error)
	validateTokenFunc func(tokenString string) (*app.Claims, error)
}

func (m *MockAuthService) Register(login, password string) (*models.AuthResponse, error) {
	if m.registerFunc != nil {
		return m.registerFunc(login, password)
	}
	return &models.AuthResponse{Token: "mock-token", UserID: "mock-user"}, nil
}

func (m *MockAuthService) Login(login, password string) (*models.AuthResponse, error) {
	if m.loginFunc != nil {
		return m.loginFunc(login, password)
	}
	return &models.AuthResponse{Token: "mock-token", UserID: "mock-user"}, nil
}

func (m *MockAuthService) ValidateToken(tokenString string) (*app.Claims, error) {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(tokenString)
	}
	return &app.Claims{UserID: "user-123", Login: "testuser"}, nil
}

// Helper functions as methods to avoid conflicts
func (h *TestHandler) writeError(w http.ResponseWriter, status int, message string) {
	// Используем ту же функцию writeError, что и в основном коде
	writeError(w, status, message)
}

func (h *TestHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	// Используем ту же функцию writeJSON, что и в основном коде
	writeJSON(w, status, data)
}

// Register handler
func (h *TestHandler) Register(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Registration request", zap.String("remote_addr", r.RemoteAddr))

	var req models.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode registration request", zap.Error(err))
		h.writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	resp, err := h.authService.Register(req.Login, req.Password)
	if err != nil {
		h.logger.Warn("Registration failed", zap.String("login", req.Login), zap.Error(err))
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("User registered successfully",
		zap.String("user_id", resp.UserID),
		zap.String("login", req.Login),
	)
	h.writeJSON(w, http.StatusOK, resp)
}

// Login handler
func (h *TestHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Login request", zap.String("remote_addr", r.RemoteAddr))

	var req models.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode login request", zap.Error(err))
		h.writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	resp, err := h.authService.Login(req.Login, req.Password)
	if err != nil {
		h.logger.Warn("Login failed", zap.String("login", req.Login), zap.Error(err))
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.logger.Info("User logged in successfully",
		zap.String("user_id", resp.UserID),
		zap.String("login", req.Login),
	)
	h.writeJSON(w, http.StatusOK, resp)
}

// AuthMiddleware копируем middleware из оригинального Handler для тестов
func (h *TestHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.logger.Warn("Authorization header missing", zap.String("path", r.URL.Path))
			writeError(w, http.StatusUnauthorized, "authorization header required")
			return
		}

		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
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

// Тесты для регистрации и логина
func TestHandler_Register(t *testing.T) {
	t.Run("should register user successfully", func(t *testing.T) {
		mockService := &MockAuthService{
			registerFunc: func(login, password string) (*models.AuthResponse, error) {
				assert.Equal(t, "testuser", login)
				assert.Equal(t, "testpass", password)
				return &models.AuthResponse{Token: "jwt-token", UserID: "user-123"}, nil
			},
		}

		handler := &TestHandler{
			authService: mockService,
			logger:      zap.NewNop(),
		}

		reqBody := models.RegisterRequest{
			Login:    "testuser",
			Password: "testpass",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp models.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "jwt-token", resp.Token)
		assert.Equal(t, "user-123", resp.UserID)
	})

	t.Run("should return bad request for invalid JSON", func(t *testing.T) {
		mockService := &MockAuthService{}

		handler := &TestHandler{
			authService: mockService,
			logger:      zap.NewNop(),
		}

		req := httptest.NewRequest("POST", "/register", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should handle registration validation error", func(t *testing.T) {
		mockService := &MockAuthService{
			registerFunc: func(login, password string) (*models.AuthResponse, error) {
				return nil, assert.AnError
			},
		}

		handler := &TestHandler{
			authService: mockService,
			logger:      zap.NewNop(),
		}

		reqBody := models.RegisterRequest{
			Login:    "testuser",
			Password: "testpass",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_Login(t *testing.T) {
	t.Run("should login user successfully", func(t *testing.T) {
		mockService := &MockAuthService{
			loginFunc: func(login, password string) (*models.AuthResponse, error) {
				assert.Equal(t, "testuser", login)
				assert.Equal(t, "testpass", password)
				return &models.AuthResponse{Token: "login-token", UserID: "user-456"}, nil
			},
		}

		handler := &TestHandler{
			authService: mockService,
			logger:      zap.NewNop(),
		}

		reqBody := models.LoginRequest{
			Login:    "testuser",
			Password: "testpass",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp models.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "login-token", resp.Token)
		assert.Equal(t, "user-456", resp.UserID)
	})

	t.Run("should return unauthorized for invalid credentials", func(t *testing.T) {
		mockService := &MockAuthService{
			loginFunc: func(login, password string) (*models.AuthResponse, error) {
				return nil, assert.AnError
			},
		}

		handler := &TestHandler{
			authService: mockService,
			logger:      zap.NewNop(),
		}

		reqBody := models.LoginRequest{
			Login:    "wronguser",
			Password: "wrongpass",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// Тесты для middleware
func TestAuthMiddleware(t *testing.T) {
	logger := zap.NewNop()

	// Создаем валидные claims для тестов
	validClaims := &app.Claims{
		UserID: "user123",
		Login:  "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	tests := []struct {
		name           string
		authHeader     string
		validateToken  func(token string) (*app.Claims, error)
		expectedStatus int
		expectedBody   string
		checkContext   func(t *testing.T, r *http.Request)
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			validateToken:  nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"authorization header required"}`,
			checkContext:   nil,
		},
		{
			name:           "missing bearer prefix",
			authHeader:     "Token abc123",
			validateToken:  nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"bearer token required"}`,
			checkContext:   nil,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid_token",
			validateToken: func(token string) (*app.Claims, error) {
				return nil, errors.New("invalid token")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid token"}`,
			checkContext:   nil,
		},
		{
			name:       "valid token",
			authHeader: "Bearer valid_token",
			validateToken: func(token string) (*app.Claims, error) {
				return validClaims, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
			checkContext: func(t *testing.T, r *http.Request) {
				userID, ok := GetUserID(r.Context())
				if !ok || userID != "user123" {
					t.Errorf("Expected userID 'user123' in context, got '%s' (ok: %t)", userID, ok)
				}

				login, ok := GetUserLogin(r.Context())
				if !ok || login != "testuser" {
					t.Errorf("Expected login 'testuser' in context, got '%s' (ok: %t)", login, ok)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := &MockAuthService{
				validateTokenFunc: tt.validateToken,
			}

			// Используем TestHandler вместо оригинального Handler
			handler := &TestHandler{
				logger:      logger,
				authService: mockAuth, // Теперь типы совместимы!
			}

			// Создаем тестовый обработчик
			var capturedRequest *http.Request
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequest = r

				if tt.checkContext != nil {
					tt.checkContext(t, r)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			middleware := handler.AuthMiddleware(testHandler)
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedBody != "" {
				// Обрезаем перевод строки в конце для сравнения
				expected := tt.expectedBody
				actual := strings.TrimSpace(rr.Body.String())
				assert.Equal(t, expected, actual, "Body mismatch")
			}

			if tt.expectedStatus != http.StatusOK && capturedRequest != nil {
				t.Error("Request should not reach handler for non-200 status codes")
			}
		})
	}
}

// Тест для проверки контекстных функций
func TestContextFunctions(t *testing.T) {
	t.Run("user ID context", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithUserID(ctx, "test123")

		userID, ok := GetUserID(ctx)
		assert.True(t, ok)
		assert.Equal(t, "test123", userID)
	})

	t.Run("user login context", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithUserLogin(ctx, "testuser")

		login, ok := GetUserLogin(ctx)
		assert.True(t, ok)
		assert.Equal(t, "testuser", login)
	})

	t.Run("missing values in context", func(t *testing.T) {
		ctx := context.Background()

		_, ok := GetUserID(ctx)
		assert.False(t, ok)

		_, ok = GetUserLogin(ctx)
		assert.False(t, ok)
	})
}

func TestAuthServiceInterface(t *testing.T) {
	t.Run("mock service should implement all methods", func(t *testing.T) {
		mockService := &MockAuthService{}

		// Test Register
		resp, err := mockService.Register("test", "password")
		assert.NoError(t, err)
		assert.Equal(t, "mock-user", resp.UserID)

		// Test Login
		resp, err = mockService.Login("test", "password")
		assert.NoError(t, err)
		assert.Equal(t, "mock-user", resp.UserID)

		// Test ValidateToken
		claims, err := mockService.ValidateToken("token")
		assert.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
	})
}
