package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// HealthResponse represents the response payload for /healthz.
type HealthResponse struct {
	Status string `json:"status"`
	App    string `json:"app"`
}

// HandleHealthz handles health check requests.
func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status: "ok",
		App:    "albionflow_backend",
	})
}

// SetupRouter initializes Chi router and registers endpoints with the provided ServerState.
func SetupRouter(state *ServerState) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/healthz", HandleHealthz)

	if state != nil {
		r.Get("/api/v1/refining/matrix", state.HandleRefiningMatrix)
		r.Get("/api/v1/arbitrage/routes", state.HandleArbitrageRoutes)
		r.Post("/api/v1/prices/refresh", state.HandlePriceRefresh)
	}

	return r
}
