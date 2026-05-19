package service_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	servicehandler "github.com/chronos-go/api/internal/handler/service"
	"github.com/chronos-go/api/internal/repository"
)

func newTestHandler() *servicehandler.Handler {
	return servicehandler.NewHandler(repository.NewInMemoryServiceRepo())
}

func TestCreate_Success(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]any{
		"name":             "Corte Masculino",
		"description":      "Corte simples com acabamento",
		"price_cents":      3500,
		"duration_minutes": 30,
		"provider_id":      "00000000-0000-0000-0000-000000000001",
	})

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
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
