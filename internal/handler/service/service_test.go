package service_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chronos-go/api/internal/domain"
	servicehandler "github.com/chronos-go/api/internal/handler/service"
	"github.com/chronos-go/api/internal/repository"
	"github.com/go-chi/chi/v5"
)

func newTestHandler() *servicehandler.Handler {
	return servicehandler.NewHandler(repository.NewInMemoryServiceRepo())
}

func newServiceRouter() *chi.Mux {
	h := newTestHandler()
	r := chi.NewRouter()
	r.Post("/services", h.Create)
	r.Get("/services", h.List)
	r.Get("/services/{id}", h.GetByID)
	r.Put("/services/{id}", h.Update)
	r.Delete("/services/{id}", h.Delete)
	return r
}

func serviceBody(name string) *bytes.Buffer {
	body, _ := json.Marshal(map[string]any{
		"name":             name,
		"description":      "Corte simples",
		"price_cents":      3500,
		"duration_minutes": 30,
		"provider_id":      "00000000-0000-0000-0000-000000000001",
	})
	return bytes.NewBuffer(body)
}

func createService(t *testing.T, r http.Handler) domain.Service {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/services", serviceBody("Corte Masculino"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var s domain.Service
	if err := json.NewDecoder(rec.Body).Decode(&s); err != nil {
		t.Fatalf("failed to decode service: %v", err)
	}
	return s
}

func TestCreate_Success(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/services", serviceBody("Corte Masculino"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", ct)
	}
}

func TestCreate_MissingName(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]any{
		"duration_minutes": 30,
		"provider_id":      "00000000-0000-0000-0000-000000000001",
	})

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreate_InvalidDuration(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]any{
		"name":             "Corte Masculino",
		"duration_minutes": 0,
		"provider_id":      "00000000-0000-0000-0000-000000000001",
	})

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreate_NegativePrice(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]any{
		"name":             "Corte Masculino",
		"price_cents":      -100,
		"duration_minutes": 30,
		"provider_id":      "00000000-0000-0000-0000-000000000001",
	})

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreate_MissingProviderID(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]any{
		"name":             "Corte Masculino",
		"duration_minutes": 30,
	})

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCreate_InvalidBody(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestList_Success(t *testing.T) {
	r := newServiceRouter()
	createService(t, r)

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	r := newServiceRouter()
	req := httptest.NewRequest(http.MethodGet, "/services/00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestUpdate_Success(t *testing.T) {
	r := newServiceRouter()
	s := createService(t, r)

	req := httptest.NewRequest(http.MethodPut, "/services/"+s.ID.String(), serviceBody("Barba"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestDelete_Success(t *testing.T) {
	r := newServiceRouter()
	s := createService(t, r)

	req := httptest.NewRequest(http.MethodDelete, "/services/"+s.ID.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
}
