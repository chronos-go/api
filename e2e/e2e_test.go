// Package e2e contains HTTP end-to-end integration tests that run against a
// real PostgreSQL instance. Tests are skipped automatically when DATABASE_URL
// is not set so that `go test ./...` remains green in environments without a
// database.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	authsvc "github.com/chronos-go/api/internal/auth"
	authhandler "github.com/chronos-go/api/internal/handler/auth"
	"github.com/chronos-go/api/internal/handler/client"
	"github.com/chronos-go/api/internal/handler/health"
	"github.com/chronos-go/api/internal/handler/provider"
	servicehandler "github.com/chronos-go/api/internal/handler/service"
	"github.com/chronos-go/api/internal/httpx"
	"github.com/chronos-go/api/internal/middleware/ratelimit"
	securitymw "github.com/chronos-go/api/internal/middleware/security"
	"github.com/chronos-go/api/internal/repository"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── TestMain ────────────────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	if os.Getenv("DATABASE_URL") == "" {
		fmt.Fprintln(os.Stderr, "e2e: DATABASE_URL not set — skipping integration tests")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// ─── testServer holds a running httptest.Server backed by real repositories ──

type testServer struct {
	srv         *httptest.Server
	pool        *pgxpool.Pool
	jwtService  *authsvc.JWTService
	rateLimiter *ratelimit.Limiter
}

// newTestServer creates a complete server identical to main.go but backed by
// the real DATABASE_URL pool. Each test should call cleanupTables(t, ts.pool).
func newTestServer(t *testing.T) *testServer {
	t.Helper()

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("pool.Ping: %v", err)
	}
	t.Cleanup(pool.Close)

	secret := jwtSecretForTest()
	jwtSvc, err := authsvc.NewJWTService(secret, "chronos-api", 60*time.Minute)
	if err != nil {
		t.Fatalf("NewJWTService: %v", err)
	}

	sessionSvc, err := authsvc.NewSessionService(jwtSvc, repository.NewSessionRepo(pool), 7*24*time.Hour)
	if err != nil {
		t.Fatalf("NewSessionService: %v", err)
	}

	serviceRepo := repository.NewServiceRepo(pool)
	providerRepo := repository.NewProviderRepo(pool)
	clientRepo := repository.NewClientRepo(pool)

	// Set the global repository defaults so that package-level functions
	// (e.g. repository.GetProviderByEmail used by the login handler) use
	// the same PostgreSQL-backed repositories as the test handlers.
	repository.SetProviderRepository(providerRepo)
	repository.SetClientRepository(clientRepo)

	loginHandler := authhandler.NewHandlerWithSessions(jwtSvc, sessionSvc)

	// Default limiter: very permissive so normal tests are not affected.
	limiter, err := ratelimit.New(1000, time.Second, false)
	if err != nil {
		t.Fatalf("ratelimit.New: %v", err)
	}

	r := buildRouter(jwtSvc, serviceRepo, providerRepo, clientRepo, loginHandler, limiter)

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return &testServer{srv: ts, pool: pool, jwtService: jwtSvc, rateLimiter: limiter}
}

// newTestServerWithLimiter creates a server with a custom rate limiter.
func newTestServerWithLimiter(t *testing.T, limit int, window time.Duration) *testServer {
	t.Helper()

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("pool.Ping: %v", err)
	}
	t.Cleanup(pool.Close)

	secret := jwtSecretForTest()
	jwtSvc, err := authsvc.NewJWTService(secret, "chronos-api", 60*time.Minute)
	if err != nil {
		t.Fatalf("NewJWTService: %v", err)
	}

	sessionSvc, err := authsvc.NewSessionService(jwtSvc, repository.NewSessionRepo(pool), 7*24*time.Hour)
	if err != nil {
		t.Fatalf("NewSessionService: %v", err)
	}

	serviceRepo := repository.NewServiceRepo(pool)
	providerRepo := repository.NewProviderRepo(pool)
	clientRepo := repository.NewClientRepo(pool)

	// Set the global repository defaults so that package-level functions
	// (e.g. repository.GetProviderByEmail used by the login handler) use
	// the same PostgreSQL-backed repositories as the test handlers.
	repository.SetProviderRepository(providerRepo)
	repository.SetClientRepository(clientRepo)

	loginHandler := authhandler.NewHandlerWithSessions(jwtSvc, sessionSvc)

	limiter, err := ratelimit.New(limit, window, false)
	if err != nil {
		t.Fatalf("ratelimit.New: %v", err)
	}

	r := buildRouter(jwtSvc, serviceRepo, providerRepo, clientRepo, loginHandler, limiter)

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return &testServer{srv: ts, pool: pool, jwtService: jwtSvc, rateLimiter: limiter}
}

func buildRouter(
	jwtSvc *authsvc.JWTService,
	serviceRepo repository.ServiceRepository,
	providerRepo repository.ProviderRepository,
	clientRepo repository.ClientRepository,
	loginHandler *authhandler.Handler,
	limiter *ratelimit.Limiter,
) *chi.Mux {
	serviceh := servicehandler.NewHandler(serviceRepo)
	clienth := client.NewHandler(clientRepo)
	providerh := provider.NewHandler(providerRepo)

	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(httpx.SecurityHeaders)
	r.Use(httpx.LimitBody(httpx.DefaultMaxBodyBytes))

	r.Get("/health", health.Get)
	r.Post("/clients", clienth.Create)
	r.Post("/providers", providerh.Register)
	r.With(limiter.Middleware).Post("/login", loginHandler.Login)
	r.Route("/auth", func(ar chi.Router) {
		ar.With(limiter.Middleware).Post("/login", loginHandler.Login)
		ar.With(limiter.Middleware).Post("/refresh", loginHandler.Refresh)
		ar.With(limiter.Middleware).Post("/logout", loginHandler.Logout)
	})

	r.Group(func(protected chi.Router) {
		protected.Use(securitymw.Authenticate(jwtSvc))
		protected.With(securitymw.RequireRoles("client")).Get("/clients/me", clienth.GetMe)
		protected.With(securitymw.RequireRoles("client")).Put("/clients/me", clienth.UpdateMe)
		protected.With(securitymw.RequireRoles("client")).Delete("/clients/me", clienth.DeleteMe)
		protected.With(securitymw.RequireRoles("provider")).Get("/providers/me", providerh.GetMe)
		protected.With(securitymw.RequireRoles("provider")).Put("/providers/me", providerh.UpdateMe)
		protected.With(securitymw.RequireRoles("provider")).Delete("/providers/me", providerh.DeleteMe)
		protected.Get("/providers", providerh.List)
		protected.Get("/providers/{id}", providerh.GetByID)
		protected.Get("/providers/{id}/details", providerh.GetDetails)
		protected.Get("/services", serviceh.List)
		protected.Get("/services/{id}", serviceh.GetByID)
		protected.With(securitymw.RequireRoles("provider")).Post("/services", serviceh.Create)

		ownsService := securitymw.RequireOwnership("id", func(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
			svc, err := serviceRepo.GetByID(id)
			return svc.ProviderID, err
		})
		protected.With(securitymw.RequireRoles("provider"), ownsService).Put("/services/{id}", serviceh.Update)
		protected.With(securitymw.RequireRoles("provider"), ownsService).Delete("/services/{id}", serviceh.Delete)
	})

	return r
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func jwtSecretForTest() string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	return "e2e-test-secret-must-be-at-least-32-chars"
}

// cleanupTables deletes all rows from every table used by the tests.
// It registers itself via t.Cleanup so it runs even if the test panics/fails.
func cleanupTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), `
			DELETE FROM auth_sessions;
			DELETE FROM services;
			DELETE FROM clients;
			DELETE FROM providers;
		`)
		if err != nil {
			t.Logf("cleanupTables: %v", err)
		}
	})
}

// do sends an HTTP request to the test server and returns the response.
// body may be nil. authToken may be empty.
func do(t *testing.T, ts *testServer, method, path string, body any, authToken string) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("do: marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, ts.srv.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("do: new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return resp
}

// readBody reads and decodes the response body into a generic map.
// It also closes the response body.
func readBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		// If body is empty (e.g. 204 No Content), return empty map.
		return map[string]any{}
	}
	return out
}

// mustStatus fails the test if the response status doesn't match.
func mustStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected status %d, got %d: %s", want, resp.StatusCode, string(body))
	}
}

// mustField returns a top-level field from a decoded JSON body and fails if
// the field is absent or has a zero value.
func mustField(t *testing.T, body map[string]any, field string) string {
	t.Helper()
	val, ok := body[field]
	if !ok {
		t.Fatalf("field %q not found in response body: %v", field, body)
	}
	s, ok := val.(string)
	if !ok || s == "" {
		t.Fatalf("field %q has unexpected value %v", field, val)
	}
	return s
}

// assertNoField fails if the field is present in the response body.
func assertNoField(t *testing.T, body map[string]any, field string) {
	t.Helper()
	if _, ok := body[field]; ok {
		t.Errorf("field %q must NOT appear in response body, but it does", field)
	}
}

// uniqueEmail generates a unique email address using a UUID suffix to avoid
// conflicts between parallel tests or repeated test runs.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s+%s@example.com", prefix, uuid.New().String()[:8])
}

// loginPayload builds the login request body.
func loginPayload(email, password, role string) map[string]string {
	return map[string]string{"email": email, "password": password, "role": role}
}

// mustRegisterProvider registers a new provider and returns its ID and tokens.
func mustRegisterProvider(t *testing.T, ts *testServer, email, password string) (id, accessToken, refreshToken string) {
	t.Helper()

	resp := do(t, ts, http.MethodPost, "/providers", map[string]string{
		"name":     "Provider Test",
		"email":    email,
		"document": "12345678000199",
		"password": password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	body := readBody(t, resp)
	id = mustField(t, body, "id")

	resp2 := do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "provider"), "")
	mustStatus(t, resp2, http.StatusOK)
	body2 := readBody(t, resp2)
	accessToken = mustField(t, body2, "access_token")
	refreshToken = mustField(t, body2, "refresh_token")

	return id, accessToken, refreshToken
}

// mustRegisterClient registers a new client and returns its ID and tokens.
func mustRegisterClient(t *testing.T, ts *testServer, email, password string) (id, accessToken, refreshToken string) {
	t.Helper()

	resp := do(t, ts, http.MethodPost, "/clients", map[string]string{
		"name":       "Client Test",
		"email":      email,
		"birth_date": "1990-01-15",
		"password":   password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	body := readBody(t, resp)
	id = mustField(t, body, "id")

	resp2 := do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "client"), "")
	mustStatus(t, resp2, http.StatusOK)
	body2 := readBody(t, resp2)
	accessToken = mustField(t, body2, "access_token")
	refreshToken = mustField(t, body2, "refresh_token")

	return id, accessToken, refreshToken
}
