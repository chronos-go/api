package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chronos-go/api/internal/domain"
	"github.com/chronos-go/api/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	repo repository.ServiceRepository
}

func NewHandler(repo repository.ServiceRepository) *Handler {
	return &Handler{repo: repo}
}

type createRequest struct {
	ProviderID      string `json:"provider_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	PriceCents      int    `json:"price_cents"`
	DurationMinutes int    `json:"duration_minutes"`
}

type errorResponse struct {
	Error string `json:"error"`
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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if strings.TrimSpace(req.ProviderID) == "" {
		writeError(w, http.StatusBadRequest, "provider_id is required")
		return
	}
	if req.DurationMinutes <= 0 {
		writeError(w, http.StatusBadRequest, "duration_minutes must be a positive integer")
		return
	}
	if req.PriceCents < 0 {
		writeError(w, http.StatusBadRequest, "price_cents cannot be negative")
		return
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider_id format")
		return
	}

	s := domain.Service{
		ID:              uuid.New(),
		ProviderID:      providerID,
		Name:            req.Name,
		Description:     req.Description,
		PriceCents:      req.PriceCents,
		DurationMinutes: req.DurationMinutes,
		CreatedAt:       time.Now(),
	}

	if err := h.repo.Save(s); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save service")
		return
	}

	writeJSON(w, http.StatusCreated, s)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	s, err := h.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrServiceNotFound) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get service")
		return
	}

	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	services, err := h.repo.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list services")
		return
	}
	writeJSON(w, http.StatusOK, services)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	if err := h.repo.Delete(id); err != nil {
		if errors.Is(err, repository.ErrServiceNotFound) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete service")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.DurationMinutes <= 0 {
		writeError(w, http.StatusBadRequest, "duration_minutes must be a positive integer")
		return
	}
	if req.PriceCents < 0 {
		writeError(w, http.StatusBadRequest, "price_cents cannot be negative")
		return
	}

	s := domain.Service{
		ID:              id,
		Name:            req.Name,
		Description:     req.Description,
		PriceCents:      req.PriceCents,
		DurationMinutes: req.DurationMinutes,
	}

	if err := h.repo.Update(s); err != nil {
		if errors.Is(err, repository.ErrServiceNotFound) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update service")
		return
	}

	writeJSON(w, http.StatusOK, s)
}

// Create é mantido para compatibilidade com os testes existentes (sem banco).
func Create(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "use handler via NewHandler")
}
