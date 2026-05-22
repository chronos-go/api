package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/chronos-go/api/internal/db"
	"github.com/chronos-go/api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProviderNotFound = errors.New("provider not found")
var ErrProviderEmailConflict = errors.New("email already registered")

type ProviderRepository interface {
	SaveProvider(p domain.Provider) error
	GetProviderByID(id uuid.UUID) (domain.Provider, error)
	GetProviderByEmail(email string) (domain.Provider, error)
	ListProviders() ([]domain.Provider, error)
	GetProviderDetails(id uuid.UUID) (ProviderDetails, error)
}

type ProviderDetails struct {
	Provider domain.Provider
	Services []domain.Service
}

var defaultProviderRepo ProviderRepository = NewInMemoryProviderRepo()

func SetProviderRepository(repo ProviderRepository) {
	if repo != nil {
		defaultProviderRepo = repo
	}
}

func SaveProvider(p domain.Provider) error {
	return defaultProviderRepo.SaveProvider(p)
}

func GetProviderByID(id uuid.UUID) (domain.Provider, error) {
	return defaultProviderRepo.GetProviderByID(id)
}

func GetProviderByEmail(email string) (domain.Provider, error) {
	return defaultProviderRepo.GetProviderByEmail(email)
}

func ListProviders() []domain.Provider {
	providers, err := defaultProviderRepo.ListProviders()
	if err != nil {
		return []domain.Provider{}
	}
	return providers
}

func GetProviderDetails(id uuid.UUID) (ProviderDetails, error) {
	return defaultProviderRepo.GetProviderDetails(id)
}

// ── PostgreSQL ────────────────────────────────────────────────────────────────

type ProviderRepo struct {
	queries *db.Queries
}

func NewProviderRepo(pool *pgxpool.Pool) *ProviderRepo {
	return &ProviderRepo{queries: db.New(pool)}
}

func (r *ProviderRepo) SaveProvider(p domain.Provider) error {
	_, err := r.queries.CreateProvider(context.Background(), db.CreateProviderParams{
		Name:     p.Name,
		Email:    p.Email,
		Document: p.Document,
		Password: p.Password,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrProviderEmailConflict
		}
		return err
	}
	return nil
}

func (r *ProviderRepo) GetProviderByID(id uuid.UUID) (domain.Provider, error) {
	provider, err := r.queries.GetProviderByID(context.Background(), toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Provider{}, ErrProviderNotFound
		}
		return domain.Provider{}, err
	}
	return toDomainProvider(provider), nil
}

func (r *ProviderRepo) GetProviderByEmail(email string) (domain.Provider, error) {
	provider, err := r.queries.GetProviderByEmail(context.Background(), email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Provider{}, ErrProviderNotFound
		}
		return domain.Provider{}, err
	}
	return toDomainProvider(provider), nil
}

func (r *ProviderRepo) ListProviders() ([]domain.Provider, error) {
	rows, err := r.queries.ListProviders(context.Background())
	if err != nil {
		return nil, err
	}
	providers := make([]domain.Provider, 0, len(rows))
	for _, row := range rows {
		providers = append(providers, toDomainProvider(row))
	}
	return providers, nil
}

func (r *ProviderRepo) GetProviderDetails(id uuid.UUID) (ProviderDetails, error) {
	provider, err := r.GetProviderByID(id)
	if err != nil {
		return ProviderDetails{}, err
	}

	rows, err := r.queries.ListServicesByProviderID(context.Background(), toPgUUID(id))
	if err != nil {
		return ProviderDetails{}, err
	}

	services := make([]domain.Service, 0, len(rows))
	for _, row := range rows {
		services = append(services, toDomainService(row))
	}
	if services == nil {
		services = []domain.Service{}
	}

	return ProviderDetails{Provider: provider, Services: services}, nil
}

// ── In-Memory (tests and fallback) ────────────────────────────────────────────

type InMemoryProviderRepo struct {
	mu   sync.RWMutex
	data map[uuid.UUID]domain.Provider
}

func NewInMemoryProviderRepo() *InMemoryProviderRepo {
	return &InMemoryProviderRepo{data: make(map[uuid.UUID]domain.Provider)}
}

func (r *InMemoryProviderRepo) SaveProvider(p domain.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.data {
		if existing.Email == p.Email {
			return ErrProviderEmailConflict
		}
	}
	r.data[p.ID] = p
	return nil
}

func (r *InMemoryProviderRepo) GetProviderByID(id uuid.UUID) (domain.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.data[id]
	if !ok {
		return domain.Provider{}, ErrProviderNotFound
	}
	return p, nil
}

func (r *InMemoryProviderRepo) GetProviderByEmail(email string) (domain.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.data {
		if p.Email == email {
			return p, nil
		}
	}
	return domain.Provider{}, ErrProviderNotFound
}

func (r *InMemoryProviderRepo) ListProviders() ([]domain.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Provider, 0, len(r.data))
	for _, p := range r.data {
		result = append(result, p)
	}
	return result, nil
}

func (r *InMemoryProviderRepo) GetProviderDetails(id uuid.UUID) (ProviderDetails, error) {
	provider, err := r.GetProviderByID(id)
	if err != nil {
		return ProviderDetails{}, err
	}
	services, err := ListServicesByProviderID(id)
	if err != nil {
		return ProviderDetails{}, err
	}
	if services == nil {
		services = []domain.Service{}
	}
	return ProviderDetails{Provider: provider, Services: services}, nil
}

func toDomainProvider(provider db.Provider) domain.Provider {
	return domain.Provider{
		ID:        fromPgUUID(provider.ID),
		Name:      provider.Name,
		Email:     provider.Email,
		Document:  provider.Document,
		Password:  provider.Password,
		CreatedAt: fromPgTime(provider.CreatedAt),
	}
}
