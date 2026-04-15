package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/chronos-go/api/internal/handler/health"
)

func main() {
	r := chi.NewRouter()

	r.Get("/health", health.Get)

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
