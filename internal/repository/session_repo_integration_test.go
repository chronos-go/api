package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	authsvc "github.com/chronos-go/api/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSessionRepo_RotationAndReplayWithPostgres(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is required for integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE auth_sessions"); err != nil {
		t.Fatal(err)
	}

	jwt, err := authsvc.NewJWTService("test-secret-with-at-least-32-characters", "chronos-api", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := authsvc.NewSessionService(jwt, NewSessionRepo(pool), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := sessions.Issue(ctx, authsvc.TokenInput{Subject: "provider-1", Role: "provider", Email: "provider@test.dev"})
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := sessions.Refresh(ctx, issued.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Refresh(ctx, issued.RefreshToken); !errors.Is(err, authsvc.ErrRefreshTokenReplay) {
		t.Fatalf("expected replay detection, got %v", err)
	}
	if _, err := sessions.Refresh(ctx, rotated.RefreshToken); !errors.Is(err, authsvc.ErrRefreshTokenReplay) {
		t.Fatalf("expected family revocation, got %v", err)
	}
}
