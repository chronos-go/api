package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/chronos-go/api/internal/handler/client"
	"github.com/chronos-go/api/internal/handler/health"
	"github.com/chronos-go/api/internal/handler/service"
)

func main() {
	r := chi.NewRouter()

	r.Get("/health", health.Get)

	r.Post("/clients", client.Create)

	r.Post("/services", service.Create)

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
