package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrExpiredRefreshToken = errors.New("refresh token expired")
	ErrRefreshTokenReplay  = errors.New("refresh token replay detected")
)

type Session struct {
	ID        uuid.UUID
	UserID    string
	Role      string
	Email     string
	FamilyID  uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type SessionStore interface {
	Create(ctx context.Context, session Session) error
	Rotate(ctx context.Context, currentHash string, replacement Session, now time.Time) (Session, error)
	Revoke(ctx context.Context, tokenHash string, now time.Time) error
}

type TokenPair struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

type SessionService struct {
	jwt        *JWTService
	store      SessionStore
	refreshTTL time.Duration
	nowFn      func() time.Time
}

func NewSessionService(jwt *JWTService, store SessionStore, refreshTTL time.Duration) (*SessionService, error) {
	if jwt == nil {
		return nil, fmt.Errorf("jwt service is required")
	}
	if store == nil {
		return nil, fmt.Errorf("session store is required")
	}
	if refreshTTL <= 0 {
		return nil, fmt.Errorf("refresh ttl must be greater than zero")
	}
	return &SessionService{jwt: jwt, store: store, refreshTTL: refreshTTL, nowFn: time.Now}, nil
}

func (s *SessionService) Issue(ctx context.Context, input TokenInput) (TokenPair, error) {
	access, accessExpiry, err := s.jwt.GenerateToken(input)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, hash, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	now := s.nowFn().UTC()
	refreshExpiry := now.Add(s.refreshTTL)
	if err := s.store.Create(ctx, Session{
		ID: uuid.New(), UserID: input.Subject, Role: input.Role, Email: input.Email,
		FamilyID: uuid.New(), TokenHash: hash, ExpiresAt: refreshExpiry, CreatedAt: now,
	}); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken: access, AccessTokenExpiresAt: accessExpiry,
		RefreshToken: refresh, RefreshTokenExpiresAt: refreshExpiry,
	}, nil
}

func (s *SessionService) Refresh(ctx context.Context, currentToken string) (TokenPair, error) {
	if currentToken == "" {
		return TokenPair{}, ErrInvalidRefreshToken
	}
	refresh, replacementHash, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	now := s.nowFn().UTC()
	replacement := Session{ID: uuid.New(), TokenHash: replacementHash, ExpiresAt: now.Add(s.refreshTTL), CreatedAt: now}
	current, err := s.store.Rotate(ctx, hashRefreshToken(currentToken), replacement, now)
	if err != nil {
		return TokenPair{}, err
	}
	access, accessExpiry, err := s.jwt.GenerateToken(TokenInput{Subject: current.UserID, Role: current.Role, Email: current.Email})
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken: access, AccessTokenExpiresAt: accessExpiry,
		RefreshToken: refresh, RefreshTokenExpiresAt: replacement.ExpiresAt,
	}, nil
}

func (s *SessionService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return ErrInvalidRefreshToken
	}
	return s.store.Revoke(ctx, hashRefreshToken(refreshToken), s.nowFn().UTC())
}

func newRefreshToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, hashRefreshToken(token), nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
