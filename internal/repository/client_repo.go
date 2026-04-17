package repository

import (
	"errors"

	"github.com/chronos-go/api/internal/domain"
	"github.com/google/uuid"
)

var ErrClientNotFound = errors.New("client not found")
var ErrClientEmailConflict = errors.New("email already registered")

var clientsDB = make(map[uuid.UUID]domain.Client)

func SaveClient(c domain.Client) error {
	for _, existing := range clientsDB {
		if existing.Email == c.Email {
			return ErrClientEmailConflict
		}
	}
	clientsDB[c.ID] = c
	return nil
}

func GetClientByID(id uuid.UUID) (domain.Client, error) {
	c, ok := clientsDB[id]
	if !ok {
		return domain.Client{}, ErrClientNotFound
	}
	return c, nil
}

func GetClientByEmail(email string) (domain.Client, error) {
	for _, c := range clientsDB {
		if c.Email == email {
			return c, nil
		}
	}
	return domain.Client{}, ErrClientNotFound
}

func ListClients() []domain.Client {
	result := make([]domain.Client, 0, len(clientsDB))
	for _, c := range clientsDB {
		result = append(result, c)
	}
	return result
}

func DeleteClient(id uuid.UUID) error {
	if _, ok := clientsDB[id]; !ok {
		return ErrClientNotFound
	}
	delete(clientsDB, id)
	return nil
}
