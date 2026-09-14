package main

import (
	"log"
	"net/http"
	"os"

	"albionflow/backend/pkg/aodp"
	"albionflow/backend/pkg/api"
	"albionflow/backend/pkg/store"

	"github.com/go-chi/chi/v5"
)

type ServerState = api.ServerState
type HealthResponse = api.HealthResponse
type RefiningMatrixItem = api.RefiningMatrixItem

func setupRouter(state *ServerState) *chi.Mux {
	return api.SetupRouter(state)
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	api.HandleHealthz(w, r)
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	s, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer s.Close()

	client := aodp.NewClient()
	state := &api.ServerState{
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
