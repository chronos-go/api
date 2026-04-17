package repository

import (
	"errors"

	"github.com/chronos-go/api/internal/domain"
	"github.com/google/uuid"
)

var ErrServiceNotFound = errors.New("service not found")

var servicesDB = make(map[uuid.UUID]domain.Service)

func SaveService(s domain.Service) error {
	servicesDB[s.ID] = s
	return nil
}

func GetServiceByID(id uuid.UUID) (domain.Service, error) {
	s, ok := servicesDB[id]
	if !ok {
		return domain.Service{}, ErrServiceNotFound
	}
	return s, nil
}

func ListServices() []domain.Service {
	result := make([]domain.Service, 0, len(servicesDB))
	for _, s := range servicesDB {
		result = append(result, s)
	}
	return result
}

func DeleteService(id uuid.UUID) error {
	if _, ok := servicesDB[id]; !ok {
		return ErrServiceNotFound
	}
	delete(servicesDB, id)
	return nil
}
