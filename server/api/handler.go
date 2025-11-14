package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/buharamanya/gophkeeper/server/app"
	"github.com/buharamanya/gophkeeper/server/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
)

// ErrorResponse стандартный формат ответа с ошибкой
type ErrorResponse struct {
	Error string `json:"error"`
}

type Handler struct {
	authService *app.AuthService
	dataService *app.DataService
	cfg         *config.Config
	logger      *zap.Logger
	server      *http.Server
}

func NewHandler(authService *app.AuthService, dataService *app.DataService, cfg *config.Config, logger *zap.Logger) *Handler {
	return &Handler{
		authService: authService,
		dataService: dataService,
		cfg:         cfg,
		logger:      logger,
	}
}

func (h *Handler) Start(address string) error {
	r := chi.NewRouter()

	// Middleware
	r.Use(h.loggingMiddleware)
	r.Use(h.recoveryMiddleware) // Добавляем middleware для обработки паник
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health check endpoint
	r.Get("/health", h.healthCheck)

	// Public routes
	r.Route("/api", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(h.AuthMiddleware)
			r.Get("/data", h.ListData)
			r.Get("/data/{id}", h.GetData)
			r.Post("/data", h.CreateData)
			r.Put("/data/{id}", h.UpdateData)
			r.Delete("/data/{id}", h.DeleteData)
			r.Post("/sync", h.Sync)
			r.Get("/sync/status", h.GetSyncStatus)
			r.Post("/sync/resolve", h.ResolveConflict)
		})
	})

	// Обработчик для 404
	r.NotFound(h.notFoundHandler)

	// Обработчик для методов не разрешенных
	r.MethodNotAllowed(h.methodNotAllowedHandler)

	// Create HTTP server with timeouts
	h.server = &http.Server{
		Addr:         address,
		Handler:      r,
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
		IdleTimeout:  60 * time.Minute,
	}

	// Load TLS configuration if enabled
	if h.cfg.EnableTLS {
		tlsConfig, err := h.cfg.LoadTLSConfig()
		if err != nil {
			h.logger.Error("Failed to load TLS config", zap.Error(err))
			return err
		}

		if tlsConfig != nil {
			h.server.TLSConfig = tlsConfig
			h.logger.Info("Starting GophKeeper server with TLS", zap.String("address", address))
			return h.server.ListenAndServeTLS("", "")
		}
	}

	h.logger.Info("Starting GophKeeper server", zap.String("address", address))
	return h.server.ListenAndServe()
}

// healthCheck обработчик для health check
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Health check requested")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// notFoundHandler обработчик для 404 ошибок
func (h *Handler) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Warn("Route not found",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote_addr", r.RemoteAddr),
	)
	writeError(w, http.StatusNotFound, "Resource not found")
}

// methodNotAllowedHandler обработчик для 405 ошибок
func (h *Handler) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Warn("Method not allowed",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote_addr", r.RemoteAddr),
	)
	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// recoveryMiddleware обрабатывает паники и преобразует их в 500 ошибки
func (h *Handler) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				h.logger.Error("Panic recovered",
					zap.Any("panic", err),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
					zap.Stack("stack"),
				)

				// Отправляем безопасный ответ без деталей ошибки
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware добавляет логирование запросов
func (h *Handler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем ResponseWriter для отслеживания статуса ответа
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		// Логируем завершение запроса
		fields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
			zap.Int("status", ww.Status()),
			zap.Int("bytes", ww.BytesWritten()),
			zap.Duration("duration", duration),
		}

		// Логируем с разным уровнем в зависимости от статуса
		switch {
		case ww.Status() >= 500:
			h.logger.Error("Server error", fields...)
		case ww.Status() >= 400:
			h.logger.Warn("Client error", fields...)
		default:
			h.logger.Info("Request completed", fields...)
		}
	})
}

// writeJSON универсальная функция для отправки JSON ответов
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Логируем ошибку кодирования, но не отправляем детали клиенту
	}
}

// writeError отправляет ошибку клиенту в безопасном формате
func writeError(w http.ResponseWriter, status int, message string) {
	// Для 5xx ошибок используем стандартные сообщения из http.StatusText
	if status >= 500 {
		message = http.StatusText(status)
	}

	response := ErrorResponse{
		Error: message,
	}
	writeJSON(w, status, response)
}

// writeInternalError отправляет 500 ошибку без деталей
func writeInternalError(w http.ResponseWriter, logger *zap.Logger, err error, context string) {
	logger.Error("Internal server error",
		zap.String("context", context),
		zap.Error(err),
	)
	writeError(w, http.StatusInternalServerError, "Internal server error")
}

// Shutdown gracefully останавливает сервер
func (h *Handler) Shutdown(ctx context.Context) error {
	h.logger.Info("Initiating graceful shutdown...")

	if h.server != nil {
		if err := h.server.Shutdown(ctx); err != nil {
			h.logger.Error("Error during server shutdown", zap.Error(err))
			return err
		}
		h.logger.Info("HTTP server shutdown completed")
	} else {
		h.logger.Warn("Server instance is nil, nothing to shutdown")
	}

	return nil
}
