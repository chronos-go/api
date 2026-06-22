package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func newTestSessionService(t *testing.T) *SessionService {
	t.Helper()
	jwt, err := NewJWTService("test-secret-with-at-least-32-characters", "chronos-api", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewSessionService(jwt, NewMemorySessionStore(), 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestSessionService_RotatesRefreshTokenAndDetectsReplay(t *testing.T) {
	service := newTestSessionService(t)
	ctx := context.Background()
	issued, err := service.Issue(ctx, TokenInput{Subject: "user-1", Role: "provider", Email: "p@test.dev"})
	if err != nil {
		t.Fatal(err)
	}
	if issued.AccessToken == "" || issued.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	rotated, err := service.Refresh(ctx, issued.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == issued.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}

	if _, err := service.Refresh(ctx, issued.RefreshToken); !errors.Is(err, ErrRefreshTokenReplay) {
		t.Fatalf("expected replay error, got %v", err)
	}
	if _, err := service.Refresh(ctx, rotated.RefreshToken); !errors.Is(err, ErrRefreshTokenReplay) {
		t.Fatalf("expected family revocation after replay, got %v", err)
	}
}

func TestSessionService_LogoutRevokesSession(t *testing.T) {
	service := newTestSessionService(t)
	ctx := context.Background()
	issued, err := service.Issue(ctx, TokenInput{Subject: "user-1", Role: "client", Email: "c@test.dev"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Logout(ctx, issued.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Refresh(ctx, issued.RefreshToken); !errors.Is(err, ErrRefreshTokenReplay) {
		t.Fatalf("expected revoked session to fail, got %v", err)
	}
}
