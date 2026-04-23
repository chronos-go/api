package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	authsvc "github.com/chronos-go/api/internal/auth"
	authhandler "github.com/chronos-go/api/internal/handler/auth"

	"github.com/go-chi/chi/v5"

	"github.com/chronos-go/api/internal/handler/client"
	"github.com/chronos-go/api/internal/handler/health"
	"github.com/chronos-go/api/internal/handler/provider"
	"github.com/chronos-go/api/internal/handler/service"
)

func main() {
	jwtService, err := authsvc.NewJWTService(jwtSecret(), jwtIssuer(), jwtTTL())
	if err != nil {
		log.Fatalf("failed to initialize auth service: %v", err)
	}
	loginHandler := authhandler.NewHandler(jwtService)

	r := chi.NewRouter()

	r.Get("/health", health.Get)
	r.Post("/clients", client.Create)
	r.Post("/providers", provider.Register)
	r.Get("/providers", provider.List)
	r.Get("/providers/{id}", provider.GetByID)
	r.Post("/services", service.Create)
	r.Post("/login", loginHandler.Login)

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func jwtSecret() string {
	if v := os.Getenv("JWT_SECRET"); v != "" {
		return v
	}
	return "dev-secret-change-me"
}

func jwtIssuer() string {
	if v := os.Getenv("JWT_ISSUER"); v != "" {
		return v
	}
	return "chronos-api"
}

func jwtTTL() time.Duration {
	minutes := 60
	if v := os.Getenv("JWT_ACCESS_TTL_MINUTES"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			minutes = parsed
		}
	}
	return time.Duration(minutes) * time.Minute
}
