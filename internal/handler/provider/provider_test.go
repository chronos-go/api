package provider_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chronos-go/api/internal/domain"
	"github.com/chronos-go/api/internal/handler/provider"
	securitymw "github.com/chronos-go/api/internal/middleware/security"
	"github.com/chronos-go/api/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func newRepo() *repository.InMemoryProviderRepo {
	return repository.NewInMemoryProviderRepo()
}

func newRouter() (*chi.Mux, *repository.InMemoryProviderRepo) {
	repo := newRepo()
	h := provider.NewHandler(repo)
	r := chi.NewRouter()
	r.Post("/providers", h.Register)
	r.Get("/providers", h.List)
	r.Get("/providers/{id}", h.GetByID)
	r.Get("/providers/{id}/details", h.GetDetails)
	return r, repo
}

func withIdentity(r *http.Request, id uuid.UUID, role string) *http.Request {
	ctx := securitymw.WithIdentity(r.Context(), securitymw.Identity{
		Subject: id.String(),
		Role:    role,
		Email:   "test@example.com",
	})
	return r.WithContext(ctx)
}

func registerBody(name, email, document, password string) *bytes.Buffer {
	b, _ := json.Marshal(map[string]string{
		"name": name, "email": email, "document": document, "password": password,
	})
	return bytes.NewBuffer(b)
}

func TestRegister_Success(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/providers", registerBody("Ana", "ana@test.com", "12345678900", "secret"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["id"] == "" {
		t.Fatal("expected non-empty id")
	}
	if _, ok := resp["password"]; ok {
		t.Fatal("password must not be exposed in response")
	}
}

func TestRegister_MissingFields(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/providers", registerBody("", "missing@test.com", "123", "pass"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/providers", registerBody("Bob", "notanemail", "99988877766", "pass"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRegister_EmailConflict(t *testing.T) {
	r, _ := newRouter()
	body := func() *bytes.Buffer {
		return registerBody("Carol", "carol@test.com", "11122233344", "pass")
	}

	req1 := httptest.NewRequest(http.MethodPost, "/providers", body())
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req1)

	rec := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/providers", body())
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req2)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestGetByID_Success(t *testing.T) {
	r, _ := newRouter()

	regReq := httptest.NewRequest(http.MethodPost, "/providers", registerBody("Dave", "dave@test.com", "55566677788", "pass"))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	r.ServeHTTP(regRec, regReq)

	var resp map[string]any
	json.NewDecoder(regRec.Body).Decode(&resp)
	id := resp["id"].(string)

	req := httptest.NewRequest(http.MethodGet, "/providers/"+id, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGetByID_InvalidUUID(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/providers/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/providers/00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestList_ReturnsArray(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/providers", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var result []any
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("expected JSON array, got error: %v", err)
	}
}

func TestGetDetails_WithServices(t *testing.T) {
	providerRepo := newRepo()
	serviceRepo := repository.NewInMemoryServiceRepo()
	h := provider.NewHandler(providerRepo)
	repository.SetServiceRepository(serviceRepo)

	r := chi.NewRouter()
	r.Get("/providers/{id}/details", h.GetDetails)

	providerID := uuid.New()
	if err := providerRepo.SaveProvider(domain.Provider{
		ID:        providerID,
		Name:      "Ana Provider",
		Email:     "ana-details@test.com",
		Document:  "12345678900",
		Password:  "hash",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("failed to seed provider: %v", err)
	}

	if _, err := serviceRepo.Create(domain.Service{
		ID:              uuid.New(),
		ProviderID:      providerID,
		Name:            "Corte Masculino",
		Description:     "Corte simples",
		PriceCents:      3500,
		DurationMinutes: 30,
		CreatedAt:       time.Now(),
	}); err != nil {
		t.Fatalf("failed to seed service: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/providers/"+providerID.String()+"/details", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	services := resp["services"].([]any)
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
}

func TestGetDetails_WithoutServices(t *testing.T) {
	providerRepo := newRepo()
	serviceRepo := repository.NewInMemoryServiceRepo()
	h := provider.NewHandler(providerRepo)
	repository.SetServiceRepository(serviceRepo)

	r := chi.NewRouter()
	r.Get("/providers/{id}/details", h.GetDetails)

	providerID := uuid.New()
	if err := providerRepo.SaveProvider(domain.Provider{
		ID:        providerID,
		Name:      "Ana Sem Serviços",
		Email:     "ana-empty@test.com",
		Document:  "12345678900",
		Password:  "hash",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("failed to seed provider: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/providers/"+providerID.String()+"/details", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	services := resp["services"].([]any)
	if len(services) != 0 {
		t.Fatalf("expected 0 services, got %d", len(services))
	}
}

func TestGetDetails_NotFound(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/providers/00000000-0000-0000-0000-000000000001/details", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetDetails_InvalidUUID(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/providers/not-a-uuid/details", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetMe_Success(t *testing.T) {
	repo := newRepo()
	h := provider.NewHandler(repo)

	id := uuid.New()
	repo.SaveProvider(domain.Provider{
		ID: id, Name: "Eve", Email: "eve@test.com", Document: "111", Password: "hash", CreatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/providers/me", nil)
	req = withIdentity(req, id, "provider")
	rec := httptest.NewRecorder()

	h.GetMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetMe_Unauthenticated(t *testing.T) {
	h := provider.NewHandler(newRepo())
	req := httptest.NewRequest(http.MethodGet, "/providers/me", nil)
	rec := httptest.NewRecorder()

	h.GetMe(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestDeleteMe_Success(t *testing.T) {
	repo := newRepo()
	h := provider.NewHandler(repo)

	id := uuid.New()
	repo.SaveProvider(domain.Provider{
		ID: id, Name: "Frank", Email: "frank@test.com", Document: "222", Password: "hash", CreatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodDelete, "/providers/me", nil)
	req = withIdentity(req, id, "provider")
	rec := httptest.NewRecorder()

	h.DeleteMe(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	_, err := repo.GetProviderByID(id)
	if err == nil {
		t.Fatal("expected provider to be deleted")
	}
}

func TestUpdateMe_Success(t *testing.T) {
	repo := newRepo()
	h := provider.NewHandler(repo)

	id := uuid.New()
	repo.SaveProvider(domain.Provider{
		ID: id, Name: "Grace", Email: "grace@test.com", Document: "333", Password: "hash", CreatedAt: time.Now(),
	})

	body, _ := json.Marshal(map[string]string{
		"name": "Grace Atualizada", "email": "grace.nova@test.com", "document": "333",
	})

	req := httptest.NewRequest(http.MethodPut, "/providers/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withIdentity(req, id, "provider")
	rec := httptest.NewRecorder()

	h.UpdateMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["name"] != "Grace Atualizada" {
		t.Fatalf("expected updated name, got %v", resp["name"])
	}
}
