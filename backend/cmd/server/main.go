package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"albionflow/backend/pkg/aodp"
	"albionflow/backend/pkg/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type HealthResponse struct {
	Status string `json:"status"`
	App    string `json:"app"`
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status: "ok",
		App:    "albionflow_backend",
	})
}

func setupRouter(state *ServerState) *chi.Mux {
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

	r.Get("/healthz", handleHealthz)

	if state != nil {
		r.Get("/api/v1/refining/matrix", state.HandleRefiningMatrix)
		r.Get("/api/v1/arbitrage/routes", state.HandleArbitrageRoutes)
		r.Post("/api/v1/prices/refresh", state.HandlePriceRefresh)
	}

	return r
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	s, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer s.Close()

	client := aodp.NewClient()
	state := &ServerState{
		Store:      s,
		AODPClient: client,
	}

	r := setupRouter(state)
	addr := ":8081"
	if envPort := os.Getenv("PORT"); envPort != "" {
		addr = ":" + envPort
	}
	log.Printf("AlbionFlow backend listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
