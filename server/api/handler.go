package api

import (
	"encoding/json"
	"net/http"

	"github.com/buharamanya/gophkeeper/server/app"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Handler struct {
	authService *app.AuthService
	dataService *app.DataService
}

func NewHandler(authService *app.AuthService, dataService *app.DataService) *Handler {
	return &Handler{
		authService: authService,
		dataService: dataService,
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

	return http.ListenAndServe(address, r)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
