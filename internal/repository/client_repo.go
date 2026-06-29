package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/chronos-go/api/internal/db"
	"github.com/chronos-go/api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrClientNotFound = errors.New("client not found")
var ErrClientEmailConflict = errors.New("email already registered")

type ClientRepository interface {
	Save(c domain.Client) (domain.Client, error)
	GetByID(id uuid.UUID) (domain.Client, error)
	GetByEmail(email string) (domain.Client, error)
	List() ([]domain.Client, error)
	Update(c domain.Client) (domain.Client, error)
	Delete(id uuid.UUID) error
}

var defaultClientRepo ClientRepository = NewInMemoryClientRepo()

func SetClientRepository(repo ClientRepository) {
	if repo != nil {
		defaultClientRepo = repo
	}
}

func SaveClient(c domain.Client) (domain.Client, error) {
	return defaultClientRepo.Save(c)
}

func GetClientByID(id uuid.UUID) (domain.Client, error) {
	return defaultClientRepo.GetByID(id)
}

func GetClientByEmail(email string) (domain.Client, error) {
	return defaultClientRepo.GetByEmail(email)
}

func ListClients() ([]domain.Client, error) {
	return defaultClientRepo.List()
}

func UpdateClient(c domain.Client) (domain.Client, error) {
	return defaultClientRepo.Update(c)
}

func DeleteClient(id uuid.UUID) error {
	return defaultClientRepo.Delete(id)
}

// ── PostgreSQL ────────────────────────────────────────────────────────────────

type ClientRepo struct {
	queries *db.Queries
}

func NewClientRepo(pool *pgxpool.Pool) *ClientRepo {
	return &ClientRepo{queries: db.New(pool)}
}

func (r *ClientRepo) Save(c domain.Client) (domain.Client, error) {
	row, err := r.queries.CreateClient(context.Background(), db.CreateClientParams{
		ID:        toPgUUID(c.ID),
		Name:      c.Name,
		Email:     c.Email,
		BirthDate: toPgDate(c.BirthDate),
		Password:  c.Password,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Client{}, ErrClientEmailConflict
		}
		return domain.Client{}, err
	}
	return toDomainClient(row), nil
}

func (r *ClientRepo) GetByID(id uuid.UUID) (domain.Client, error) {
	row, err := r.queries.GetClientByID(context.Background(), toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Client{}, ErrClientNotFound
		}
		return domain.Client{}, err
	}
	return toDomainClient(row), nil
}

func (r *ClientRepo) GetByEmail(email string) (domain.Client, error) {
	row, err := r.queries.GetClientByEmail(context.Background(), email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Client{}, ErrClientNotFound
		}
		return domain.Client{}, err
	}
	return toDomainClient(row), nil
}

func (r *ClientRepo) List() ([]domain.Client, error) {
	rows, err := r.queries.ListClients(context.Background())
	if err != nil {
		return nil, err
	}
	clients := make([]domain.Client, 0, len(rows))
	for _, row := range rows {
		clients = append(clients, toDomainClient(row))
	}
	return clients, nil
}

func (r *ClientRepo) Update(c domain.Client) (domain.Client, error) {
	row, err := r.queries.UpdateClient(context.Background(), db.UpdateClientParams{
		ID:        toPgUUID(c.ID),
		Name:      c.Name,
		Email:     c.Email,
		BirthDate: toPgDate(c.BirthDate),
		Password:  c.Password,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Client{}, ErrClientNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Client{}, ErrClientEmailConflict
		}
		return domain.Client{}, err
	}
	return toDomainClient(row), nil
}

func (r *ClientRepo) Delete(id uuid.UUID) error {
	rows, err := r.queries.DeleteClient(context.Background(), toPgUUID(id))
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrClientNotFound
	}
	return nil
}

// ── In-Memory (test double) ───────────────────────────────────────────────────

type InMemoryClientRepo struct {
	mu   sync.RWMutex
	data map[uuid.UUID]domain.Client
}

func NewInMemoryClientRepo() *InMemoryClientRepo {
	return &InMemoryClientRepo{data: make(map[uuid.UUID]domain.Client)}
}

func (r *InMemoryClientRepo) Save(c domain.Client) (domain.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.data {
		if existing.Email == c.Email {
			return domain.Client{}, ErrClientEmailConflict
		}
	}
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	r.data[c.ID] = c
	return c, nil
}

func (r *InMemoryClientRepo) GetByID(id uuid.UUID) (domain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.data[id]
	if !ok {
		return domain.Client{}, ErrClientNotFound
	}
	return c, nil
}

func (r *InMemoryClientRepo) GetByEmail(email string) (domain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.data {
		if c.Email == email {
			return c, nil
		}
	}
	return domain.Client{}, ErrClientNotFound
}

func (r *InMemoryClientRepo) List() ([]domain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Client, 0, len(r.data))
	for _, c := range r.data {
		result = append(result, c)
	}
	return result, nil
}

func (r *InMemoryClientRepo) Update(c domain.Client) (domain.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[c.ID]; !ok {
		return domain.Client{}, ErrClientNotFound
	}
	for _, existing := range r.data {
		if existing.Email == c.Email && existing.ID != c.ID {
			return domain.Client{}, ErrClientEmailConflict
		}
	}
	r.data[c.ID] = c
	return c, nil
}

func (r *InMemoryClientRepo) Delete(id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[id]; !ok {
		return ErrClientNotFound
	}
	delete(r.data, id)
	return nil
}

// ── Converters ────────────────────────────────────────────────────────────────

func toDomainClient(c db.Client) domain.Client {
	return domain.Client{
		ID:        fromPgUUID(c.ID),
		Name:      c.Name,
		Email:     c.Email,
		BirthDate: fromPgDate(c.BirthDate),
		Password:  c.Password,
		CreatedAt: fromPgTime(c.CreatedAt),
	}
}

func toPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func fromPgDate(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}
