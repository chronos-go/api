package domain

import (
	"time"

	"github.com/google/uuid"
)

type Provider struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Document  string    `json:"document"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
