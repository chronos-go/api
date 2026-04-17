package repository

import (
	"errors"

	"github.com/chronos-go/api/internal/domain"
	"github.com/google/uuid"
)

var ErrProviderNotFound = errors.New("provider not found")
var ErrProviderEmailConflict = errors.New("email already registered")

var providersDB = make(map[uuid.UUID]domain.Provider)

func SaveProvider(p domain.Provider) error {
	for _, existing := range providersDB {
		if existing.Email == p.Email {
			return ErrProviderEmailConflict
		}
	}
	providersDB[p.ID] = p
	return nil
}

func GetProviderByID(id uuid.UUID) (domain.Provider, error) {
	p, ok := providersDB[id]
	if !ok {
		return domain.Provider{}, ErrProviderNotFound
	}
	return p, nil
}

func GetProviderByEmail(email string) (domain.Provider, error) {
	for _, p := range providersDB {
		if p.Email == email {
			return p, nil
		}
	}
	return domain.Provider{}, ErrProviderNotFound
}

func ListProviders() []domain.Provider {
	result := make([]domain.Provider, 0, len(providersDB))
	for _, p := range providersDB {
		result = append(result, p)
	}
	return result
}

func DeleteProvider(id uuid.UUID) error {
	if _, ok := providersDB[id]; !ok {
		return ErrProviderNotFound
	}
	delete(providersDB, id)
	return nil
}
