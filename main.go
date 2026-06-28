package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	authsvc "github.com/chronos-go/api/internal/auth"
	authhandler "github.com/chronos-go/api/internal/handler/auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/chronos-go/api/internal/database"
	"github.com/chronos-go/api/internal/handler/client"
	"github.com/chronos-go/api/internal/handler/health"
	"github.com/chronos-go/api/internal/handler/provider"
	servicehandler "github.com/chronos-go/api/internal/handler/service"
	"github.com/chronos-go/api/internal/httpx"
	"github.com/chronos-go/api/internal/middleware/ratelimit"
	securitymw "github.com/chronos-go/api/internal/middleware/security"
	"github.com/chronos-go/api/internal/repository"
)

func main() {
	db := database.Connect()
	defer db.Close()

	serviceRepo := repository.NewServiceRepo(db)
	providerRepo := repository.NewProviderRepo(db)
	clientRepo := repository.NewClientRepo(db)
	repository.SetServiceRepository(serviceRepo)
	repository.SetProviderRepository(providerRepo)
	repository.SetClientRepository(clientRepo)
	serviceh := servicehandler.NewHandler(serviceRepo)
	clienth := client.NewHandler(clientRepo)

	secret := jwtSecret()
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET must be set and contain at least 32 characters")
	}
	jwtService, err := authsvc.NewJWTService(secret, jwtIssuer(), jwtTTL())
	if err != nil {
		log.Fatalf("failed to initialize auth service: %v", err)
	}
	sessionService, err := authsvc.NewSessionService(jwtService, repository.NewSessionRepo(db), refreshTTL())
	if err != nil {
		log.Fatalf("failed to initialize session service: %v", err)
	}
	loginHandler := authhandler.NewHandlerWithSessions(jwtService, sessionService)
	authLimiter, err := ratelimit.New(rateLimitRequests(), rateLimitWindow(), trustProxy())
	if err != nil {
		log.Fatalf("failed to initialize rate limiter: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(httpx.SecurityHeaders)
	r.Use(httpx.LimitBody(httpx.DefaultMaxBodyBytes))
	r.Use(httpx.CORS(httpx.CORSConfig{AllowedOrigins: allowedOrigins()}))

	r.Get("/health", health.Get)
	r.Post("/clients", clienth.Create)
	r.Post("/providers", provider.Register)
	r.With(authLimiter.Middleware).Post("/login", loginHandler.Login)
	r.Route("/auth", func(ar chi.Router) {
		ar.With(authLimiter.Middleware).Post("/login", loginHandler.Login)
		ar.With(authLimiter.Middleware).Post("/refresh", loginHandler.Refresh)
		ar.With(authLimiter.Middleware).Post("/logout", loginHandler.Logout)
	})

	r.Group(func(protected chi.Router) {
		protected.Use(securitymw.Authenticate(jwtService))
		protected.With(securitymw.RequireRoles("client")).Get("/clients/me", clienth.GetMe)
		protected.With(securitymw.RequireRoles("client")).Put("/clients/me", clienth.UpdateMe)
		protected.With(securitymw.RequireRoles("client")).Delete("/clients/me", clienth.DeleteMe)
		protected.Get("/providers", provider.List)
		protected.Get("/providers/{id}", provider.GetByID)
		protected.Get("/providers/{id}/details", provider.GetDetails)
		protected.Get("/services", serviceh.List)
		protected.Get("/services/{id}", serviceh.GetByID)
		protected.With(securitymw.RequireRoles("provider")).Post("/services", serviceh.Create)

		ownsService := securitymw.RequireOwnership("id", func(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
			service, err := serviceRepo.GetByID(id)
			return service.ProviderID, err
		})
		protected.With(securitymw.RequireRoles("provider"), ownsService).Put("/services/{id}", serviceh.Update)
		protected.With(securitymw.RequireRoles("provider"), ownsService).Delete("/services/{id}", serviceh.Delete)
	})

	server := httpx.NewServer(":8080", r)
	errCh := make(chan error, 1)
	go func() {
		log.Println("server listening on :8080")
		errCh <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("server shutdown failed: %v", err)
		}
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}
}

func jwtSecret() string {
	return os.Getenv("JWT_SECRET")
}

func refreshTTL() time.Duration {
	days := positiveEnvInt("JWT_REFRESH_TTL_DAYS", 7)
	return time.Duration(days) * 24 * time.Hour
}

func rateLimitRequests() int { return positiveEnvInt("AUTH_RATE_LIMIT_REQUESTS", 10) }

func rateLimitWindow() time.Duration {
	seconds := positiveEnvInt("AUTH_RATE_LIMIT_WINDOW_SECONDS", 60)
	return time.Duration(seconds) * time.Second
}

func positiveEnvInt(name string, fallback int) int {
	if value := os.Getenv(name); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func trustProxy() bool {
	value, _ := strconv.ParseBool(os.Getenv("TRUST_PROXY"))
	return value
}

func allowedOrigins() []string {
	if value := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); value != "" {
		return strings.Split(value, ",")
	}
	return nil
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
