package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewJWTService_ValidateInput(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		issuer string
		ttl    time.Duration
	}{
		{name: "missing secret", secret: "", issuer: "chronos-api", ttl: time.Minute},
		{name: "missing issuer", secret: "secret", issuer: "", ttl: time.Minute},
		{name: "invalid ttl", secret: "secret", issuer: "chronos-api", ttl: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewJWTService(tt.secret, tt.issuer, tt.ttl)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestGenerateAndValidateToken_Success(t *testing.T) {
	svc, err := NewJWTService("super-secret", "chronos-api", 30*time.Minute)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	fixedNow := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	svc.nowFn = func() time.Time { return fixedNow }

	token, expiresAt, err := svc.GenerateToken(TokenInput{
		Subject: "user-123",
		Role:    "provider",
		Email:   "ana@chronos.dev",
		Provisional: map[string]any{
			"session_version": "v1",
			"auth_source":     "login",
		},
	})
	if err != nil {
		t.Fatalf("unexpected generate error: %v", err)
	}

	if want := fixedNow.Add(30 * time.Minute); !expiresAt.Equal(want) {
		t.Fatalf("unexpected expiresAt: got %v want %v", expiresAt, want)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}

	if claims.Subject != "user-123" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
	if claims.Role != "provider" {
		t.Fatalf("unexpected role: %s", claims.Role)
	}
	if claims.Email != "ana@chronos.dev" {
		t.Fatalf("unexpected email: %s", claims.Email)
	}
	if claims.Provisional["session_version"] != "v1" {
		t.Fatalf("unexpected provisional claim: %v", claims.Provisional["session_version"])
	}
}

func TestGenerateToken_RequiresSubject(t *testing.T) {
	svc, err := NewJWTService("super-secret", "chronos-api", time.Minute)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	_, _, err = svc.GenerateToken(TokenInput{})
	if err == nil {
		t.Fatal("expected subject validation error")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	svc, err := NewJWTService("super-secret", "chronos-api", time.Minute)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	now := time.Now().UTC()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			Issuer:    "chronos-api",
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	encoded, err := token.SignedString([]byte("super-secret"))
	if err != nil {
		t.Fatalf("unexpected sign error: %v", err)
	}

	_, err = svc.ValidateToken(encoded)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	svc, err := NewJWTService("expected-secret", "chronos-api", time.Minute)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	now := time.Now().UTC()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			Issuer:    "chronos-api",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	encoded, err := token.SignedString([]byte("other-secret"))
	if err != nil {
		t.Fatalf("unexpected sign error: %v", err)
	}

	_, err = svc.ValidateToken(encoded)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}
