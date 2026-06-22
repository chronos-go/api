package repository

import (
	"context"
	"errors"
	"time"

	authsvc "github.com/chronos-go/api/internal/auth"
	"github.com/chronos-go/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool, queries: db.New(pool)}
}

func (r *SessionRepo) Create(ctx context.Context, session authsvc.Session) error {
	_, err := r.queries.CreateAuthSession(ctx, toCreateAuthSessionParams(session))
	return err
}

func (r *SessionRepo) Rotate(ctx context.Context, currentHash string, replacement authsvc.Session, now time.Time) (authsvc.Session, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return authsvc.Session{}, err
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)

	row, err := queries.GetAuthSessionForUpdate(ctx, currentHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return authsvc.Session{}, authsvc.ErrInvalidRefreshToken
	}
	if err != nil {
		return authsvc.Session{}, err
	}
	current := toDomainSession(row)
	revokedAt := pgtype.Timestamptz{Time: now, Valid: true}
	if current.UsedAt != nil || current.RevokedAt != nil {
		if err := queries.RevokeAuthSessionFamily(ctx, db.RevokeAuthSessionFamilyParams{FamilyID: row.FamilyID, RevokedAt: revokedAt}); err != nil {
			return authsvc.Session{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return authsvc.Session{}, err
		}
		return authsvc.Session{}, authsvc.ErrRefreshTokenReplay
	}
	if !current.ExpiresAt.After(now) {
		if err := queries.RevokeAuthSessionFamily(ctx, db.RevokeAuthSessionFamilyParams{FamilyID: row.FamilyID, RevokedAt: revokedAt}); err != nil {
			return authsvc.Session{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return authsvc.Session{}, err
		}
		return authsvc.Session{}, authsvc.ErrExpiredRefreshToken
	}

	rows, err := queries.MarkAuthSessionUsed(ctx, db.MarkAuthSessionUsedParams{ID: row.ID, UsedAt: pgtype.Timestamptz{Time: now, Valid: true}})
	if err != nil {
		return authsvc.Session{}, err
	}
	if rows != 1 {
		return authsvc.Session{}, authsvc.ErrRefreshTokenReplay
	}
	replacement.UserID = current.UserID
	replacement.Role = current.Role
	replacement.Email = current.Email
	replacement.FamilyID = current.FamilyID
	if _, err := queries.CreateAuthSession(ctx, toCreateAuthSessionParams(replacement)); err != nil {
		return authsvc.Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return authsvc.Session{}, err
	}
	return current, nil
}

func (r *SessionRepo) Revoke(ctx context.Context, tokenHash string, now time.Time) error {
	row, err := r.queries.GetAuthSessionByHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return authsvc.ErrInvalidRefreshToken
	}
	if err != nil {
		return err
	}
	return r.queries.RevokeAuthSessionFamily(ctx, db.RevokeAuthSessionFamilyParams{
		FamilyID: row.FamilyID, RevokedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
}

func toCreateAuthSessionParams(session authsvc.Session) db.CreateAuthSessionParams {
	return db.CreateAuthSessionParams{
		ID: toPgUUID(session.ID), UserID: session.UserID, Role: session.Role, Email: session.Email,
		FamilyID: toPgUUID(session.FamilyID), TokenHash: session.TokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: session.ExpiresAt, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: session.CreatedAt, Valid: true},
	}
}

func toDomainSession(row db.AuthSession) authsvc.Session {
	session := authsvc.Session{
		ID: fromPgUUID(row.ID), UserID: row.UserID, Role: row.Role, Email: row.Email,
		FamilyID: fromPgUUID(row.FamilyID), TokenHash: row.TokenHash,
		ExpiresAt: fromPgTime(row.ExpiresAt), CreatedAt: fromPgTime(row.CreatedAt),
	}
	if row.UsedAt.Valid {
		value := row.UsedAt.Time
		session.UsedAt = &value
	}
	if row.RevokedAt.Valid {
		value := row.RevokedAt.Time
		session.RevokedAt = &value
	}
	return session
}
