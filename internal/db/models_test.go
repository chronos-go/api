package db_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/chronos-go/api/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func uuidFrom(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

func TestServiceModel_JSONContainsProviderID(t *testing.T) {
	providerID := uuidFrom("550e8400-e29b-41d4-a716-446655440001")
	serviceID := uuidFrom("550e8400-e29b-41d4-a716-446655440002")

	s := db.Service{
		ID:              serviceID,
		ProviderID:      providerID,
		Name:            "Corte",
		Description:     "desc",
		PriceCents:      1500,
		DurationMinutes: 30,
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	for _, key := range []string{"id", "provider_id", "name", "description", "price_cents", "duration_minutes", "created_at"} {
		if !strings.Contains(string(data), `"`+key+`"`) {
			t.Errorf("expected JSON key %q not found in: %s", key, data)
		}
	}
}

func TestServiceModel_ProviderIDValue(t *testing.T) {
	providerID := uuidFrom("550e8400-e29b-41d4-a716-446655440001")

	s := db.Service{ProviderID: providerID}

	data, _ := json.Marshal(s)

	var m map[string]any
	json.Unmarshal(data, &m)

	got, ok := m["provider_id"]
	if !ok {
		t.Fatal("provider_id key missing from JSON")
	}
	if got != "550e8400-e29b-41d4-a716-446655440001" {
		t.Errorf("expected provider_id UUID string, got %v", got)
	}
}

func TestProviderModel_JSONTags(t *testing.T) {
	p := db.Provider{
		ID:       uuidFrom("550e8400-e29b-41d4-a716-446655440003"),
		Name:     "Maria",
		Email:    "maria@example.com",
		Document: "12345678900",
		Password: "hashed",
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	for _, key := range []string{"id", "name", "email", "document", "password", "created_at"} {
		if !strings.Contains(string(data), `"`+key+`"`) {
			t.Errorf("expected JSON key %q not found in: %s", key, data)
		}
	}
}

func TestCreateServiceParams_JSONTags(t *testing.T) {
	params := db.CreateServiceParams{
		ProviderID:      uuidFrom("550e8400-e29b-41d4-a716-446655440001"),
		Name:            "Pedicure",
		Description:     "Cuidado dos pés",
		PriceCents:      4000,
		DurationMinutes: 50,
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	for _, key := range []string{"provider_id", "name", "description", "price_cents", "duration_minutes"} {
		if !strings.Contains(string(data), `"`+key+`"`) {
			t.Errorf("expected JSON key %q not found", key)
		}
	}
}

func TestUpdateServiceParams_JSONTags(t *testing.T) {
	params := db.UpdateServiceParams{
		ID:              uuidFrom("550e8400-e29b-41d4-a716-446655440002"),
		Name:            "Corte atualizado",
		Description:     "Nova desc",
		PriceCents:      2000,
		DurationMinutes: 35,
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	for _, key := range []string{"id", "name", "description", "price_cents", "duration_minutes"} {
		if !strings.Contains(string(data), `"`+key+`"`) {
			t.Errorf("expected JSON key %q not found", key)
		}
	}
}
