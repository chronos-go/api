package security

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	authsvc "github.com/chronos-go/api/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type identityKey struct{}

type Identity struct {
	Subject string
	Role    string
	Email   string
	TokenID string
}

func Authenticate(jwt *authsvc.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Fields(r.Header.Get("Authorization"))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			claims, err := jwt.ValidateToken(parts[1])
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired access token")
				return
			}
			identity := Identity{Subject: claims.Subject, Role: claims.Role, Email: claims.Email, TokenID: claims.ID}
			ctx := context.WithValue(r.Context(), identityKey{}, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}

func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFromContext(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if _, ok := allowed[identity.Role]; !ok {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type OwnerResolver func(ctx context.Context, resourceID uuid.UUID) (uuid.UUID, error)

func RequireOwnership(urlParam string, resolve OwnerResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFromContext(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			resourceID, err := uuid.Parse(chi.URLParam(r, urlParam))
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid resource id")
				return
			}
			ownerID, err := resolve(r.Context(), resourceID)
			if err != nil || ownerID.String() != identity.Subject {
				writeError(w, http.StatusForbidden, "resource access denied")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
