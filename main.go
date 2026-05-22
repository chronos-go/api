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
	"github.com/go-chi/chi/v5/middleware"

	"github.com/chronos-go/api/internal/database"
	"github.com/chronos-go/api/internal/handler/client"
	"github.com/chronos-go/api/internal/handler/health"
	"github.com/chronos-go/api/internal/handler/provider"
	servicehandler "github.com/chronos-go/api/internal/handler/service"
	"github.com/chronos-go/api/internal/repository"
)

func main() {
	db := database.Connect()
	defer db.Close()

	serviceRepo := repository.NewServiceRepo(db)
	providerRepo := repository.NewProviderRepo(db)
	repository.SetServiceRepository(serviceRepo)
	repository.SetProviderRepository(providerRepo)
	serviceh := servicehandler.NewHandler(serviceRepo)

	jwtService, err := authsvc.NewJWTService(jwtSecret(), jwtIssuer(), jwtTTL())
	if err != nil {
		log.Fatalf("failed to initialize auth service: %v", err)
	}
	loginHandler := authhandler.NewHandler(jwtService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", health.Get)
	r.Post("/clients", client.Create)
	r.Post("/providers", provider.Register)
	r.Get("/providers", provider.List)
	r.Get("/providers/{id}", provider.GetByID)
	r.Get("/providers/{id}/details", provider.GetDetails)
	r.Post("/login", loginHandler.Login)

	r.Post("/services", serviceh.Create)
	r.Get("/services", serviceh.List)
	r.Get("/services/{id}", serviceh.GetByID)
	r.Put("/services/{id}", serviceh.Update)
	r.Delete("/services/{id}", serviceh.Delete)

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
