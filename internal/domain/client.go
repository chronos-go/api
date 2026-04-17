package domain

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	BirthDate time.Time `json:"birth_date"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
