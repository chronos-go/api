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
	"github.com/chronos-go/api/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Document string `json:"document"`
	Password string `json:"password"`
}

type errorResponse struct {
	Error string `json:"error"`
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

func Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" ||
		strings.TrimSpace(req.Document) == "" || strings.TrimSpace(req.Password) == "" {
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

	if err := repository.SaveProvider(p); err != nil {
		if errors.Is(err, repository.ErrProviderEmailConflict) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to save provider")
		return
	}

	writeJSON(w, http.StatusCreated, p)
}

func GetByID(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	p, err := repository.GetProviderByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get provider")
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func List(w http.ResponseWriter, r *http.Request) {
	providers := repository.ListProviders()
	if providers == nil {
		providers = []domain.Provider{}
	}
	writeJSON(w, http.StatusOK, providers)
}

func GetDetails(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	details, err := repository.GetProviderDetails(id)
	if err != nil {
		if errors.Is(err, repository.ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get provider details")
		return
	}

	if details.Services == nil {
		details.Services = []domain.Service{}
	}

	writeJSON(w, http.StatusOK, providerDetailsResponse{
		ID:        details.Provider.ID.String(),
		Name:      details.Provider.Name,
		Email:     details.Provider.Email,
		Document:  details.Provider.Document,
		CreatedAt: details.Provider.CreatedAt,
		Services:  details.Services,
	})
}
