package service_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chronos-go/api/internal/handler/service"
)

func TestCreate_Success(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"name":             "Corte Masculino",
		"description":      "Corte simples com acabamento",
		"price_cents":      3500,
		"duration_minutes": 30,
		"provider_id":      "prov-123",
	})

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	service.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", ct)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", resp["status"])
	}
}

func TestCreate_MissingName(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"duration_minutes": 30,
		"provider_id":      "prov-123",
	})

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	service.Create(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", rec.Code)
	}
}

func TestCreate_InvalidDuration(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"name":             "Corte Masculino",
		"duration_minutes": 0,
		"provider_id":      "prov-123",
	})

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	service.Create(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", rec.Code)
	}
}

func TestCreate_NegativePrice(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"name":             "Corte Masculino",
		"price_cents":      -100,
		"duration_minutes": 30,
		"provider_id":      "prov-123",
	})

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	service.Create(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", rec.Code)
	}
}

func TestCreate_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	service.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}
