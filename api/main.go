package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/chronos-go/api/internal/handler/health"
	"github.com/chronos-go/api/internal/handler/provider"
)

func main() {
	r := chi.NewRouter()

	r.Get("/health", health.Get)

	r.Route("/providers", func(r chi.Router) {
		r.Post("/", provider.Register)
		r.Get("/", provider.List)
		r.Get("/{id}", provider.GetByID)
	})

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
