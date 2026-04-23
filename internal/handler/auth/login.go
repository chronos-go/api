package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	authsvc "github.com/chronos-go/api/internal/auth"
	"github.com/chronos-go/api/internal/crypto"
	"github.com/chronos-go/api/internal/repository"
)

const (
	roleClient   = "client"
	roleProvider = "provider"
)

type Handler struct {
	jwt *authsvc.JWTService
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginResponse struct {
	TokenType   string       `json:"token_type"`
	AccessToken string       `json:"access_token"`
	ExpiresIn   int64        `json:"expires_in"`
	ExpiresAt   string       `json:"expires_at"`
	User        loginUserDTO `json:"user"`
}

type loginUserDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler(jwtService *authsvc.JWTService) *Handler {
	return &Handler{jwt: jwtService}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))

	if req.Email == "" || strings.TrimSpace(req.Password) == "" || req.Role == "" {
		writeError(w, http.StatusBadRequest, "email, password and role are required")
		return
	}

	if req.Role != roleClient && req.Role != roleProvider {
		writeError(w, http.StatusBadRequest, "role must be 'client' or 'provider'")
		return
	}

	var (
		tokenInput authsvc.TokenInput
		userDTO    loginUserDTO
		passwordDB string
	)

	switch req.Role {
	case roleProvider:
		provider, err := repository.GetProviderByEmail(req.Email)
		if err != nil {
			if errors.Is(err, repository.ErrProviderNotFound) {
				writeError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to authenticate")
			return
		}

		passwordDB = provider.Password
		tokenInput = authsvc.TokenInput{
			Subject: provider.ID.String(),
			Role:    roleProvider,
			Email:   provider.Email,
			Provisional: map[string]any{
				"session_version": "v1",
				"auth_source":     "login",
			},
		}
		userDTO = loginUserDTO{ID: provider.ID.String(), Name: provider.Name, Email: provider.Email, Role: roleProvider}
	case roleClient:
		client, err := repository.GetClientByEmail(req.Email)
		if err != nil {
			if errors.Is(err, repository.ErrClientNotFound) {
				writeError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to authenticate")
			return
		}

		passwordDB = client.Password
		tokenInput = authsvc.TokenInput{
			Subject: client.ID.String(),
			Role:    roleClient,
			Email:   client.Email,
			Provisional: map[string]any{
				"session_version": "v1",
				"auth_source":     "login",
			},
		}
		userDTO = loginUserDTO{ID: client.ID.String(), Name: client.Name, Email: client.Email, Role: roleClient}
	}

	if !crypto.Compare(passwordDB, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	accessToken, expiresAt, err := h.jwt.GenerateToken(tokenInput)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		TokenType:   "Bearer",
		AccessToken: accessToken,
		ExpiresIn:   int64(time.Until(expiresAt).Seconds()),
		ExpiresAt:   expiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		User:        userDTO,
	})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
