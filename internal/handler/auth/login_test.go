package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authsvc "github.com/chronos-go/api/internal/auth"
	"github.com/chronos-go/api/internal/crypto"
	"github.com/chronos-go/api/internal/domain"
	"github.com/chronos-go/api/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func newRouter(t *testing.T) *chi.Mux {
	t.Helper()

	jwtService, err := authsvc.NewJWTService("test-secret", "chronos-api", time.Minute)
	if err != nil {
		t.Fatalf("failed to create jwt service: %v", err)
	}

	h := NewHandler(jwtService)
	r := chi.NewRouter()
	r.Post("/login", h.Login)
	return r
}

func TestLogin_SuccessProvider(t *testing.T) {
	r := newRouter(t)

	hash, err := crypto.Hash("secret123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	email := "provider-login-success@test.com"
	if err := repository.SaveProvider(domain.Provider{
		ID:        uuid.New(),
		Name:      "Ana Provider",
		Email:     email,
		Document:  "11122233344",
		Password:  hash,
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("failed to seed provider: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": "secret123",
		"role":     "provider",
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["token_type"] != "Bearer" {
		t.Fatalf("expected token_type Bearer, got %v", resp["token_type"])
	}
	if resp["access_token"] == "" {
		t.Fatal("expected access token in response")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	r := newRouter(t)

	hash, err := crypto.Hash("secret123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	email := "provider-invalid-pass@test.com"
	if err := repository.SaveProvider(domain.Provider{
		ID:        uuid.New(),
		Name:      "Bob Provider",
		Email:     email,
		Document:  "99988877766",
		Password:  hash,
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("failed to seed provider: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": "wrong",
		"role":     "provider",
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestLogin_InvalidRole(t *testing.T) {
	r := newRouter(t)

	body, _ := json.Marshal(map[string]string{
		"email":    "role@test.com",
		"password": "secret123",
		"role":     "admin",
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	r := newRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
