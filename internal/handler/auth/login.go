package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	authsvc "github.com/chronos-go/api/internal/auth"
	"github.com/chronos-go/api/internal/crypto"
	"github.com/chronos-go/api/internal/httpx"
	"github.com/chronos-go/api/internal/repository"
)

const (
	roleClient   = "client"
	roleProvider = "provider"
)

type Handler struct {
	sessions *authsvc.SessionService
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginResponse struct {
	TokenType        string       `json:"token_type"`
	AccessToken      string       `json:"access_token"`
	RefreshToken     string       `json:"refresh_token"`
	ExpiresIn        int64        `json:"expires_in"`
	ExpiresAt        string       `json:"expires_at"`
	RefreshExpiresAt string       `json:"refresh_expires_at"`
	User             loginUserDTO `json:"user"`
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
	sessions, err := authsvc.NewSessionService(jwtService, authsvc.NewMemorySessionStore(), 7*24*time.Hour)
	if err != nil {
		panic(err)
	}
	return &Handler{sessions: sessions}
}

func NewHandlerWithSessions(_ *authsvc.JWTService, sessions *authsvc.SessionService) *Handler {
	return &Handler{sessions: sessions}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
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

	tokens, err := h.sessions.Issue(r.Context(), tokenInput)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, loginResponse{
		TokenType:        "Bearer",
		AccessToken:      tokens.AccessToken,
		RefreshToken:     tokens.RefreshToken,
		ExpiresIn:        int64(time.Until(tokens.AccessTokenExpiresAt).Seconds()),
		ExpiresAt:        tokens.AccessTokenExpiresAt.UTC().Format(time.RFC3339),
		RefreshExpiresAt: tokens.RefreshTokenExpiresAt.UTC().Format(time.RFC3339),
		User:             userDTO,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	TokenType        string `json:"token_type"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int64  `json:"expires_in"`
	ExpiresAt        string `json:"expires_at"`
	RefreshExpiresAt string `json:"refresh_expires_at"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}
	tokens, err := h.sessions.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, authsvc.ErrInvalidRefreshToken) || errors.Is(err, authsvc.ErrExpiredRefreshToken) || errors.Is(err, authsvc.ErrRefreshTokenReplay) {
			writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to refresh session")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, tokenResponse{
		TokenType: "Bearer", AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken,
		ExpiresIn:        int64(time.Until(tokens.AccessTokenExpiresAt).Seconds()),
		ExpiresAt:        tokens.AccessTokenExpiresAt.UTC().Format(time.RFC3339),
		RefreshExpiresAt: tokens.RefreshTokenExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}
	if err := h.sessions.Logout(r.Context(), req.RefreshToken); err != nil && !errors.Is(err, authsvc.ErrInvalidRefreshToken) {
		writeError(w, http.StatusInternalServerError, "failed to end session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
