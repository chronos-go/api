package provider_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chronos-go/api/internal/domain"
	"github.com/chronos-go/api/internal/handler/provider"
	"github.com/go-chi/chi/v5"
)

func newRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/providers", provider.Register)
	r.Get("/providers", provider.List)
	r.Get("/providers/{id}", provider.GetByID)
	return r
}

func registerBody(name, email, document, password string) *bytes.Buffer {
	b, _ := json.Marshal(map[string]string{
		"name": name, "email": email, "document": document, "password": password,
	})
	return bytes.NewBuffer(b)
}

func TestRegister_Success(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/providers", registerBody("Ana", "ana@test.com", "12345678900", "secret"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var p domain.Provider
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if p.ID.String() == "" {
		t.Fatal("expected non-empty ID")
	}
	if p.Password != "" {
		t.Fatal("password must not be exposed in response")
	}
}

func TestRegister_MissingFields(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/providers", registerBody("", "missing@test.com", "123", "pass"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/providers", registerBody("Bob", "notanemail", "99988877766", "pass"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRegister_EmailConflict(t *testing.T) {
	r := newRouter()
	body := func() *bytes.Buffer {
		return registerBody("Carol", "carol@test.com", "11122233344", "pass")
	}

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/providers", body()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers", body())
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestGetByID_Success(t *testing.T) {
	r := newRouter()

	regReq := httptest.NewRequest(http.MethodPost, "/providers", registerBody("Dave", "dave@test.com", "55566677788", "pass"))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	r.ServeHTTP(regRec, regReq)

	var p domain.Provider
	json.NewDecoder(regRec.Body).Decode(&p)

	req := httptest.NewRequest(http.MethodGet, "/providers/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGetByID_InvalidUUID(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/providers/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/providers/00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestList_ReturnsArray(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/providers", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var result []domain.Provider
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("expected JSON array, got error: %v", err)
	}
}
