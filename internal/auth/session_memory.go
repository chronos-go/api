package auth

import (
	"context"
	"sync"
	"time"
)

type MemorySessionStore struct {
	mu       sync.Mutex
	sessions map[string]Session
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{sessions: make(map[string]Session)}
}

func (s *MemorySessionStore) Create(_ context.Context, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.TokenHash] = session
	return nil
}

func (s *MemorySessionStore) Rotate(_ context.Context, currentHash string, replacement Session, now time.Time) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[currentHash]
	if !ok {
		return Session{}, ErrInvalidRefreshToken
	}
	if current.UsedAt != nil || current.RevokedAt != nil {
		for hash, candidate := range s.sessions {
			if candidate.FamilyID == current.FamilyID {
				t := now
				candidate.RevokedAt = &t
				s.sessions[hash] = candidate
			}
		}
		return Session{}, ErrRefreshTokenReplay
	}
	if !current.ExpiresAt.After(now) {
		return Session{}, ErrExpiredRefreshToken
	}
	usedAt := now
	current.UsedAt = &usedAt
	s.sessions[currentHash] = current
	replacement.UserID = current.UserID
	replacement.Role = current.Role
	replacement.Email = current.Email
	replacement.FamilyID = current.FamilyID
	s.sessions[replacement.TokenHash] = replacement
	return current, nil
}

func (s *MemorySessionStore) Revoke(_ context.Context, tokenHash string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[tokenHash]
	if !ok {
		return ErrInvalidRefreshToken
	}
	for hash, candidate := range s.sessions {
		if candidate.FamilyID == current.FamilyID {
			t := now
			candidate.RevokedAt = &t
			s.sessions[hash] = candidate
		}
	}
	return nil
}
