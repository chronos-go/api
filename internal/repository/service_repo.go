package repository

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/chronos-go/api/internal/domain"
	"github.com/google/uuid"
)

var ErrServiceNotFound = errors.New("service not found")

// ServiceRepository define o contrato para acesso a dados de services.
type ServiceRepository interface {
	Save(s domain.Service) error
	GetByID(id uuid.UUID) (domain.Service, error)
	List() ([]domain.Service, error)
	Update(s domain.Service) error
	Delete(id uuid.UUID) error
	ListByProviderID(providerID uuid.UUID) ([]domain.Service, error)
}

// ── PostgreSQL ────────────────────────────────────────────────────────────────

type ServiceRepo struct {
	db *sql.DB
}

func NewServiceRepo(db *sql.DB) *ServiceRepo {
	return &ServiceRepo{db: db}
}

func (r *ServiceRepo) Save(s domain.Service) error {
	_, err := r.db.ExecContext(context.Background(),
		`INSERT INTO services (id, provider_id, name, description, price_cents, duration_minutes, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.ProviderID, s.Name, s.Description, s.PriceCents, s.DurationMinutes, s.CreatedAt,
	)
	return err
}

func (r *ServiceRepo) GetByID(id uuid.UUID) (domain.Service, error) {
	var s domain.Service
	err := r.db.QueryRowContext(context.Background(),
		`SELECT id, provider_id, name, description, price_cents, duration_minutes, created_at
		 FROM services WHERE id = $1`, id,
	).Scan(&s.ID, &s.ProviderID, &s.Name, &s.Description, &s.PriceCents, &s.DurationMinutes, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Service{}, ErrServiceNotFound
	}
	return s, err
}

func (r *ServiceRepo) List() ([]domain.Service, error) {
	rows, err := r.db.QueryContext(context.Background(),
		`SELECT id, provider_id, name, description, price_cents, duration_minutes, created_at
		 FROM services ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []domain.Service
	for rows.Next() {
		var s domain.Service
		if err := rows.Scan(&s.ID, &s.ProviderID, &s.Name, &s.Description, &s.PriceCents, &s.DurationMinutes, &s.CreatedAt); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	if services == nil {
		services = []domain.Service{}
	}
	return services, rows.Err()
}

func (r *ServiceRepo) Update(s domain.Service) error {
	res, err := r.db.ExecContext(context.Background(),
		`UPDATE services SET name=$1, description=$2, price_cents=$3, duration_minutes=$4
		 WHERE id=$5`,
		s.Name, s.Description, s.PriceCents, s.DurationMinutes, s.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrServiceNotFound
	}
	return nil
}

func (r *ServiceRepo) Delete(id uuid.UUID) error {
	res, err := r.db.ExecContext(context.Background(),
		`DELETE FROM services WHERE id = $1`, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrServiceNotFound
	}
	return nil
}

func (r *ServiceRepo) ListByProviderID(providerID uuid.UUID) ([]domain.Service, error) {
	rows, err := r.db.QueryContext(context.Background(),
		`SELECT id, provider_id, name, description, price_cents, duration_minutes, created_at
		 FROM services WHERE provider_id = $1 ORDER BY created_at DESC`, providerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []domain.Service
	for rows.Next() {
		var s domain.Service
		if err := rows.Scan(&s.ID, &s.ProviderID, &s.Name, &s.Description, &s.PriceCents, &s.DurationMinutes, &s.CreatedAt); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	if services == nil {
		services = []domain.Service{}
	}
	return services, rows.Err()
}

// ── In-Memory (somente para testes) ──────────────────────────────────────────

type InMemoryServiceRepo struct {
	mu   sync.RWMutex
	data map[uuid.UUID]domain.Service
}

func NewInMemoryServiceRepo() *InMemoryServiceRepo {
	return &InMemoryServiceRepo{data: make(map[uuid.UUID]domain.Service)}
}

func (r *InMemoryServiceRepo) Save(s domain.Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[s.ID] = s
	return nil
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

func (r *InMemoryServiceRepo) Update(s domain.Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[s.ID]; !ok {
		return ErrServiceNotFound
	}
	r.data[s.ID] = s
	return nil
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

var _ = time.Now
