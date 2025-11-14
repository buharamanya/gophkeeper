package api

import (
	"encoding/json"
	"net/http"

	"github.com/buharamanya/gophkeeper/server/app"
	"github.com/buharamanya/gophkeeper/server/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
)

type Handler struct {
	authService *app.AuthService
	dataService *app.DataService
	cfg         *config.Config
	logger      *zap.Logger
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

	// Middleware с логированием
	r.Use(h.loggingMiddleware)
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
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		h.logger.Debug("Health check requested")
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

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

	// Create HTTP server
	server := &http.Server{
		Addr:         address,
		Handler:      r,
		ReadTimeout:  15 * 60,
		WriteTimeout: 15 * 60,
		IdleTimeout:  60 * 60,
	}

	// Load TLS configuration if enabled
	if h.cfg.EnableTLS {
		tlsConfig, err := h.cfg.LoadTLSConfig()
		if err != nil {
			h.logger.Error("Failed to load TLS config", zap.Error(err))
			return err
		}

		if tlsConfig != nil {
			server.TLSConfig = tlsConfig
			h.logger.Info("Starting GophKeeper server with TLS", zap.String("address", address))
			return server.ListenAndServeTLS("", "")
		}
	}

	h.logger.Info("Starting GophKeeper server", zap.String("address", address))
	return server.ListenAndServe()
}

// loggingMiddleware добавляет логирование запросов
func (h *Handler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.Debug("Incoming request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Логирование ошибок будет через перехватчик паники
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
