package domain

import (
	"time"

	"github.com/google/uuid"
)

type Service struct {
	ID              uuid.UUID `json:"id"`
	ProviderID      uuid.UUID `json:"provider_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	PriceCents      int       `json:"price_cents"`
	DurationMinutes int       `json:"duration_minutes"`
	CreatedAt       time.Time `json:"created_at"`
}
