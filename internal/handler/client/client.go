package client

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
	"github.com/google/uuid"
)

type Handler struct {
	repo repository.ClientRepository
}

func NewHandler(repo repository.ClientRepository) *Handler {
	return &Handler{repo: repo}
}

type createRequest struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	BirthDate string `json:"birth_date"`
	Password  string `json:"password"`
}

type updateRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	BirthDate       string `json:"birth_date"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type clientResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	BirthDate string    `json:"birth_date"`
	CreatedAt time.Time `json:"created_at"`
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

func toResponse(c domain.Client) clientResponse {
	return clientResponse{
		ID:        c.ID.String(),
		Name:      c.Name,
		Email:     c.Email,
		BirthDate: c.BirthDate.Format("2006-01-02"),
		CreatedAt: c.CreatedAt,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.BirthDate = strings.TrimSpace(req.BirthDate)

	if req.Name == "" || req.Email == "" || req.BirthDate == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, email, birth_date and password are required")
		return
	}
	if len(req.Name) < 2 {
		writeError(w, http.StatusBadRequest, "name must be at least 2 characters")
		return
	}
	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	birthDate, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "birth_date must be in YYYY-MM-DD format")
		return
	}

	hashed, err := crypto.Hash(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	created, err := h.repo.Save(domain.Client{
		Name:      req.Name,
		Email:     req.Email,
		BirthDate: birthDate,
		Password:  hashed,
	})
	if err != nil {
		if errors.Is(err, repository.ErrClientEmailConflict) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create client")
		return
	}

	writeJSON(w, http.StatusCreated, toResponse(created))
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

	c, err := h.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			writeError(w, http.StatusNotFound, "client not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get client")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(c))
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
	req.BirthDate = strings.TrimSpace(req.BirthDate)

	if req.Name == "" || req.Email == "" || req.BirthDate == "" {
		writeError(w, http.StatusBadRequest, "name, email and birth_date are required")
		return
	}
	if len(req.Name) < 2 {
		writeError(w, http.StatusBadRequest, "name must be at least 2 characters")
		return
	}
	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	birthDate, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "birth_date must be in YYYY-MM-DD format")
		return
	}

	current, err := h.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			writeError(w, http.StatusNotFound, "client not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get client")
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
		if len(req.NewPassword) < 8 {
			writeError(w, http.StatusBadRequest, "new_password must be at least 8 characters")
			return
		}
		newPasswordHash, err = crypto.Hash(req.NewPassword)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to process password")
			return
		}
	}

	updated, err := h.repo.Update(domain.Client{
		ID:        id,
		Name:      req.Name,
		Email:     req.Email,
		BirthDate: birthDate,
		Password:  newPasswordHash,
		CreatedAt: current.CreatedAt,
	})
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			writeError(w, http.StatusNotFound, "client not found")
			return
		}
		if errors.Is(err, repository.ErrClientEmailConflict) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update client")
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

	if err := h.repo.Delete(id); err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			writeError(w, http.StatusNotFound, "client not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete client")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
