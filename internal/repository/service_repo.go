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

var ErrServiceNotFound = errors.New("service not found")
var ErrServiceProviderNotFound = errors.New("provider not found")

var defaultServiceRepo ServiceRepository = NewInMemoryServiceRepo()

func SetServiceRepository(repo ServiceRepository) {
	if repo != nil {
		defaultServiceRepo = repo
	}
}

func SaveService(s domain.Service) (domain.Service, error) {
	return defaultServiceRepo.Create(s)
}

func GetServiceByID(id uuid.UUID) (domain.Service, error) {
	return defaultServiceRepo.GetByID(id)
}

func ListServices() ([]domain.Service, error) {
	return defaultServiceRepo.List()
}

func UpdateService(s domain.Service) (domain.Service, error) {
	return defaultServiceRepo.Update(s)
}

func DeleteService(id uuid.UUID) error {
	return defaultServiceRepo.Delete(id)
}

func ListServicesByProviderID(providerID uuid.UUID) ([]domain.Service, error) {
	return defaultServiceRepo.ListByProviderID(providerID)
}

// ServiceRepository define o contrato para acesso a dados de services.
type ServiceRepository interface {
	Create(s domain.Service) (domain.Service, error)
	GetByID(id uuid.UUID) (domain.Service, error)
	List() ([]domain.Service, error)
	Update(s domain.Service) (domain.Service, error)
	Delete(id uuid.UUID) error
	ListByProviderID(providerID uuid.UUID) ([]domain.Service, error)
}

// ── PostgreSQL ────────────────────────────────────────────────────────────────

type ServiceRepo struct {
	queries *db.Queries
}

func NewServiceRepo(pool *pgxpool.Pool) *ServiceRepo {
	return &ServiceRepo{queries: db.New(pool)}
}

func (r *ServiceRepo) Create(s domain.Service) (domain.Service, error) {
	created, err := r.queries.CreateService(context.Background(), db.CreateServiceParams{
		ProviderID:      toPgUUID(s.ProviderID),
		Name:            s.Name,
		Description:     s.Description,
		PriceCents:      int32(s.PriceCents),
		DurationMinutes: int32(s.DurationMinutes),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.Service{}, ErrServiceProviderNotFound
		}
		return domain.Service{}, err
	}
	return toDomainService(created), nil
}

func (r *ServiceRepo) GetByID(id uuid.UUID) (domain.Service, error) {
	row, err := r.queries.GetServiceByID(context.Background(), toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Service{}, ErrServiceNotFound
		}
		return domain.Service{}, err
	}
	return toDomainService(row), nil
}

func (r *ServiceRepo) List() ([]domain.Service, error) {
	rows, err := r.queries.ListServices(context.Background())
	if err != nil {
		return nil, err
	}
	services := make([]domain.Service, 0, len(rows))
	for _, row := range rows {
		services = append(services, toDomainService(row))
	}
	return services, nil
}

func (r *ServiceRepo) Update(s domain.Service) (domain.Service, error) {
	updated, err := r.queries.UpdateService(context.Background(), db.UpdateServiceParams{
		ID:              toPgUUID(s.ID),
		Name:            s.Name,
		Description:     s.Description,
		PriceCents:      int32(s.PriceCents),
		DurationMinutes: int32(s.DurationMinutes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Service{}, ErrServiceNotFound
		}
		return domain.Service{}, err
	}
	return toDomainService(updated), nil
}

func (r *ServiceRepo) Delete(id uuid.UUID) error {
	if _, err := r.GetByID(id); err != nil {
		return err
	}
	return r.queries.DeleteService(context.Background(), toPgUUID(id))
}

func (r *ServiceRepo) ListByProviderID(providerID uuid.UUID) ([]domain.Service, error) {
	rows, err := r.queries.ListServicesByProviderID(context.Background(), toPgUUID(providerID))
	if err != nil {
		return nil, err
	}
	services := make([]domain.Service, 0, len(rows))
	for _, row := range rows {
		services = append(services, toDomainService(row))
	}
	return services, nil
}

// ── In-Memory (somente para testes) ──────────────────────────────────────────

type InMemoryServiceRepo struct {
	mu   sync.RWMutex
	data map[uuid.UUID]domain.Service
}

func NewInMemoryServiceRepo() *InMemoryServiceRepo {
	return &InMemoryServiceRepo{data: make(map[uuid.UUID]domain.Service)}
}

func (r *InMemoryServiceRepo) Create(s domain.Service) (domain.Service, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	r.data[s.ID] = s
	return s, nil
}

func (r *InMemoryServiceRepo) GetByID(id uuid.UUID) (domain.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.data[id]
	if !ok {
		return domain.Service{}, ErrServiceNotFound
	}
	return s, nil
}

func (r *InMemoryServiceRepo) List() ([]domain.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Service, 0, len(r.data))
	for _, s := range r.data {
		result = append(result, s)
	}
	return result, nil
}

func (r *InMemoryServiceRepo) Update(s domain.Service) (domain.Service, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.data[s.ID]
	if !ok {
		return domain.Service{}, ErrServiceNotFound
	}
	s.ProviderID = current.ProviderID
	s.CreatedAt = current.CreatedAt
	r.data[s.ID] = s
	return s, nil
}

func (r *InMemoryServiceRepo) Delete(id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[id]; !ok {
		return ErrServiceNotFound
	}
	delete(r.data, id)
	return nil
}

func (r *InMemoryServiceRepo) ListByProviderID(providerID uuid.UUID) ([]domain.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []domain.Service
	for _, s := range r.data {
		if s.ProviderID == providerID {
			result = append(result, s)
		}
	}
	if result == nil {
		result = []domain.Service{}
	}
	return result, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toDomainService(svc db.Service) domain.Service {
	return domain.Service{
		ID:              fromPgUUID(svc.ID),
		ProviderID:      fromPgUUID(svc.ProviderID),
		Name:            svc.Name,
		Description:     svc.Description,
		PriceCents:      int(svc.PriceCents),
		DurationMinutes: int(svc.DurationMinutes),
		CreatedAt:       fromPgTime(svc.CreatedAt),
	}
}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

func fromPgTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}
