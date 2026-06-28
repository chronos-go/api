package client_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chronos-go/api/internal/domain"
	"github.com/chronos-go/api/internal/handler/client"
	securitymw "github.com/chronos-go/api/internal/middleware/security"
	"github.com/chronos-go/api/internal/repository"
	"github.com/google/uuid"
)

func newHandler() *client.Handler {
	return client.NewHandler(repository.NewInMemoryClientRepo())
}

func withIdentity(r *http.Request, id uuid.UUID, role string) *http.Request {
	ctx := securitymw.WithIdentity(r.Context(), securitymw.Identity{
		Subject: id.String(),
		Role:    role,
		Email:   "test@example.com",
	})
	return r.WithContext(ctx)
}

func TestCreate_Success(t *testing.T) {
	h := newHandler()
	body, _ := json.Marshal(map[string]string{
		"name":       "Maria Silva",
		"email":      "maria@email.com",
		"birth_date": "1990-05-15",
		"password":   "secret123",
	})

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if _, ok := resp["id"]; !ok {
		t.Fatal("expected id in response")
	}
	if resp["email"] != "maria@email.com" {
		t.Fatalf("expected email maria@email.com, got %v", resp["email"])
	}
	if _, ok := resp["password"]; ok {
		t.Fatal("password must not be exposed in response")
	}
}

func TestCreate_MissingFields(t *testing.T) {
	h := newHandler()
	body, _ := json.Marshal(map[string]string{"name": "Maria Silva"})

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreate_InvalidEmail(t *testing.T) {
	h := newHandler()
	body, _ := json.Marshal(map[string]string{
		"name":       "Maria Silva",
		"email":      "not-an-email",
		"birth_date": "1990-05-15",
		"password":   "secret123",
	})

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreate_InvalidBirthDate(t *testing.T) {
	h := newHandler()
	body, _ := json.Marshal(map[string]string{
		"name":       "Maria Silva",
		"email":      "maria@email.com",
		"birth_date": "15/05/1990",
		"password":   "secret123",
	})

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreate_ShortPassword(t *testing.T) {
	h := newHandler()
	body, _ := json.Marshal(map[string]string{
		"name":       "Maria Silva",
		"email":      "maria@email.com",
		"birth_date": "1990-05-15",
		"password":   "short",
	})

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreate_DuplicateEmail(t *testing.T) {
	h := newHandler()
	body, _ := json.Marshal(map[string]string{
		"name":       "Maria Silva",
		"email":      "dup@email.com",
		"birth_date": "1990-05-15",
		"password":   "secret123",
	})

	for i := range 2 {
		req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.Create(rec, req)
		if i == 0 && rec.Code != http.StatusCreated {
			t.Fatalf("first create: expected 201, got %d", rec.Code)
		}
		if i == 1 && rec.Code != http.StatusConflict {
			t.Fatalf("second create: expected 409, got %d", rec.Code)
		}
	}
}

func TestCreate_InvalidBody(t *testing.T) {
	h := newHandler()
	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetMe_Success(t *testing.T) {
	repo := repository.NewInMemoryClientRepo()
	h := client.NewHandler(repo)

	saved, _ := repo.Save(domain.Client{
		ID:        uuid.New(),
		Name:      "João",
		Email:     "joao@email.com",
		BirthDate: time.Date(1995, 3, 10, 0, 0, 0, 0, time.UTC),
		Password:  "hashed",
		CreatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/clients/me", nil)
	req = withIdentity(req, saved.ID, "client")
	rec := httptest.NewRecorder()

	h.GetMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["email"] != "joao@email.com" {
		t.Fatalf("expected email joao@email.com, got %v", resp["email"])
	}
}

func TestGetMe_Unauthenticated(t *testing.T) {
	h := newHandler()
	req := httptest.NewRequest(http.MethodGet, "/clients/me", nil)
	rec := httptest.NewRecorder()

	h.GetMe(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestDeleteMe_Success(t *testing.T) {
	repo := repository.NewInMemoryClientRepo()
	h := client.NewHandler(repo)

	saved, _ := repo.Save(domain.Client{
		ID:        uuid.New(),
		Name:      "Ana",
		Email:     "ana@email.com",
		BirthDate: time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC),
		Password:  "hashed",
		CreatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodDelete, "/clients/me", nil)
	req = withIdentity(req, saved.ID, "client")
	rec := httptest.NewRecorder()

	h.DeleteMe(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	_, err := repo.GetByID(saved.ID)
	if err == nil {
		t.Fatal("expected client to be deleted")
	}
}

func TestUpdateMe_Success(t *testing.T) {
	repo := repository.NewInMemoryClientRepo()
	h := client.NewHandler(repo)

	saved, _ := repo.Save(domain.Client{
		ID:        uuid.New(),
		Name:      "Carlos",
		Email:     "carlos@email.com",
		BirthDate: time.Date(1988, 7, 20, 0, 0, 0, 0, time.UTC),
		Password:  "hashed",
		CreatedAt: time.Now(),
	})

	body, _ := json.Marshal(map[string]string{
		"name":       "Carlos Atualizado",
		"email":      "carlos.novo@email.com",
		"birth_date": "1988-07-20",
	})

	req := httptest.NewRequest(http.MethodPut, "/clients/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withIdentity(req, saved.ID, "client")
	rec := httptest.NewRecorder()

	h.UpdateMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["name"] != "Carlos Atualizado" {
		t.Fatalf("expected updated name, got %v", resp["name"])
	}
}
