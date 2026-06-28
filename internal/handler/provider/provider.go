package provider

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chronos-go/api/internal/crypto"
	"github.com/chronos-go/api/internal/domain"
	"github.com/chronos-go/api/internal/httpx"
	securitymw "github.com/chronos-go/api/internal/middleware/security"
	"github.com/chronos-go/api/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	repo repository.ProviderRepository
}

func NewHandler(repo repository.ProviderRepository) *Handler {
	return &Handler{repo: repo}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Document string `json:"document"`
	Password string `json:"password"`
}

type updateRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Document        string `json:"document"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type providerResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Document  string    `json:"document"`
	CreatedAt time.Time `json:"created_at"`
}

type providerDetailsResponse struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Email     string           `json:"email"`
	Document  string           `json:"document"`
	CreatedAt time.Time        `json:"created_at"`
	Services  []domain.Service `json:"services"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func toResponse(p domain.Provider) providerResponse {
	return providerResponse{
		ID:        p.ID.String(),
		Name:      p.Name,
		Email:     p.Email,
		Document:  p.Document,
		CreatedAt: p.CreatedAt,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Document = strings.TrimSpace(req.Document)

	if req.Name == "" || req.Email == "" || req.Document == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, email, document and password are required")
		return
	}
	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	hashed, err := crypto.Hash(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	p := domain.Provider{
		ID:        uuid.New(),
		Name:      req.Name,
		Email:     req.Email,
		Document:  req.Document,
		Password:  hashed,
		CreatedAt: time.Now(),
	}

	if err := h.repo.SaveProvider(p); err != nil {
		if errors.Is(err, repository.ErrProviderEmailConflict) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to save provider")
		return
	}

	writeJSON(w, http.StatusCreated, toResponse(p))
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	p, err := h.repo.GetProviderByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get provider")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(p))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	providers, err := h.repo.ListProviders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list providers")
		return
	}
	if providers == nil {
		providers = []domain.Provider{}
	}
	result := make([]providerResponse, 0, len(providers))
	for _, p := range providers {
		result = append(result, toResponse(p))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetDetails(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	details, err := h.repo.GetProviderDetails(id)
	if err != nil {
		if errors.Is(err, repository.ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get provider details")
		return
	}

	services := details.Services
	if services == nil {
		services = []domain.Service{}
	}

	writeJSON(w, http.StatusOK, providerDetailsResponse{
		ID:        details.Provider.ID.String(),
		Name:      details.Provider.Name,
		Email:     details.Provider.Email,
		Document:  details.Provider.Document,
		CreatedAt: details.Provider.CreatedAt,
		Services:  services,
	})
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	identity, ok := securitymw.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := uuid.Parse(identity.Subject)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token subject")
		return
	}

	p, err := h.repo.GetProviderByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get provider")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(p))
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	identity, ok := securitymw.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := uuid.Parse(identity.Subject)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token subject")
		return
	}

	var req updateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Document = strings.TrimSpace(req.Document)

	if req.Name == "" || req.Email == "" || req.Document == "" {
		writeError(w, http.StatusBadRequest, "name, email and document are required")
		return
	}
	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	current, err := h.repo.GetProviderByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get provider")
		return
	}

	newPasswordHash := current.Password
	if req.CurrentPassword != "" || req.NewPassword != "" {
		if req.CurrentPassword == "" || req.NewPassword == "" {
			writeError(w, http.StatusBadRequest, "both current_password and new_password are required to change password")
			return
		}
		if !crypto.Compare(current.Password, req.CurrentPassword) {
			writeError(w, http.StatusUnauthorized, "current password is incorrect")
			return
		}
		newPasswordHash, err = crypto.Hash(req.NewPassword)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to process password")
			return
		}
	}

	updated, err := h.repo.UpdateProvider(domain.Provider{
		ID:        id,
		Name:      req.Name,
		Email:     req.Email,
		Document:  req.Document,
		Password:  newPasswordHash,
		CreatedAt: current.CreatedAt,
	})
	if err != nil {
		if errors.Is(err, repository.ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		if errors.Is(err, repository.ErrProviderEmailConflict) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update provider")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(updated))
}

func (h *Handler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	identity, ok := securitymw.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := uuid.Parse(identity.Subject)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token subject")
		return
	}

	if err := h.repo.DeleteProvider(id); err != nil {
		if errors.Is(err, repository.ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete provider")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Register e demais funções globais mantidas para compatibilidade com main.go até refatoração completa.
func Register(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "use handler via NewHandler")
}

func GetByID(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "use handler via NewHandler")
}

func List(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "use handler via NewHandler")
}

func GetDetails(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "use handler via NewHandler")
}
