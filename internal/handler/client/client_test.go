package client_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chronos-go/api/internal/handler/client"
)

func TestCreate_Success(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"name":  "Maria Silva",
		"email": "maria@email.com",
		"phone": "84999990000",
	})

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	client.Create(rec, req)

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

func TestCreate_MissingFields(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"name": "Maria Silva",
	})

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	client.Create(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", rec.Code)
	}
}

func TestCreate_InvalidEmail(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"name":  "Maria Silva",
		"email": "not-an-email",
		"phone": "84999990000",
	})

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	client.Create(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", rec.Code)
	}
}

func TestCreate_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	client.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}
