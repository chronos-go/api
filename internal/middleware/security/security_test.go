package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authsvc "github.com/chronos-go/api/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func testToken(t *testing.T, subject, role string) (*authsvc.JWTService, string) {
	t.Helper()
	jwt, err := authsvc.NewJWTService("test-secret-with-at-least-32-characters", "chronos-api", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := jwt.GenerateToken(authsvc.TokenInput{Subject: subject, Role: role})
	if err != nil {
		t.Fatal(err)
	}
	return jwt, token
}

func TestAuthenticate_RequiresValidBearerToken(t *testing.T) {
	jwt, token := testToken(t, uuid.NewString(), "provider")
	handler := Authenticate(jwt)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := IdentityFromContext(r.Context()); !ok {
			t.Fatal("identity missing from context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/", nil))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", missing.Code)
	}

	validReq := httptest.NewRequest(http.MethodGet, "/", nil)
	validReq.Header.Set("Authorization", "Bearer "+token)
	valid := httptest.NewRecorder()
	handler.ServeHTTP(valid, validReq)
	if valid.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", valid.Code)
	}
}

func TestRequireRoles_DeniesWrongRole(t *testing.T) {
	jwt, token := testToken(t, uuid.NewString(), "client")
	handler := Authenticate(jwt)(RequireRoles("provider")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequireOwnership(t *testing.T) {
	ownerID := uuid.New()
	resourceID := uuid.New()
	jwt, token := testToken(t, ownerID.String(), "provider")
	router := chi.NewRouter()
	router.Use(Authenticate(jwt))
	router.With(RequireOwnership("id", func(context.Context, uuid.UUID) (uuid.UUID, error) {
		return ownerID, nil
	})).Delete("/services/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodDelete, "/services/"+resourceID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected owner access, got %d", rec.Code)
	}
}

func TestRequireOwnership_DeniesDifferentOwner(t *testing.T) {
	jwt, token := testToken(t, uuid.NewString(), "provider")
	router := chi.NewRouter()
	router.Use(Authenticate(jwt))
	router.With(RequireOwnership("id", func(context.Context, uuid.UUID) (uuid.UUID, error) {
		return uuid.New(), nil
	})).Delete("/services/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodDelete, "/services/"+uuid.NewString(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
