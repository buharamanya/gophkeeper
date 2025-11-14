package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/buharamanya/gophkeeper/server/app"
	"github.com/buharamanya/gophkeeper/server/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Handler struct {
	authService *app.AuthService
	dataService *app.DataService
	cfg         *config.Config
}

func NewHandler(authService *app.AuthService, dataService *app.DataService, cfg *config.Config) *Handler {
	return &Handler{
		authService: authService,
		dataService: dataService,
		cfg:         cfg,
	}
}

func (h *Handler) Start(address string) error {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
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
		})
	})

	// Create HTTP server with timeouts
	server := &http.Server{
		Addr:         address,
		Handler:      r,
		ReadTimeout:  15 * 60, // 15 minutes
		WriteTimeout: 15 * 60, // 15 minutes
		IdleTimeout:  60 * 60, // 1 hour
	}

	// Load TLS configuration if enabled
	if h.cfg.EnableTLS {
		tlsConfig, err := h.cfg.LoadTLSConfig()
		if err != nil {
			return err
		}

		if tlsConfig != nil {
			server.TLSConfig = tlsConfig
			log.Printf("Starting GophKeeper server with TLS on %s", address)
			return server.ListenAndServeTLS("", "")
		}
	}

	log.Printf("Starting GophKeeper server on %s", address)
	return server.ListenAndServe()
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
