package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chronos-go/api/internal/domain"
	"github.com/google/uuid"
)

func TestService_JSONMarshal_ContainsExpectedKeys(t *testing.T) {
	s := domain.Service{
		ID:              uuid.New(),
		ProviderID:      uuid.New(),
		Name:            "Corte Masculino",
		Description:     "Corte simples",
		PriceCents:      3500,
		DurationMinutes: 30,
		CreatedAt:       time.Now(),
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map failed: %v", err)
	}

	for _, key := range []string{"id", "provider_id", "name", "description", "price_cents", "duration_minutes", "created_at"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected JSON key %q not found", key)
		}
	}
}

func TestService_JSONUnmarshal_NewFields(t *testing.T) {
	providerID := uuid.New()

	raw := `{
		"name":             "Hidratacao",
		"description":      "Tratamento capilar",
		"price_cents":      5000,
		"duration_minutes": 60,
		"provider_id":      "` + providerID.String() + `"
	}`

	var s domain.Service
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if s.Name != "Hidratacao" {
		t.Errorf("expected Name %q, got %q", "Hidratacao", s.Name)
	}
	if s.PriceCents != 5000 {
		t.Errorf("expected PriceCents 5000, got %d", s.PriceCents)
	}
	if s.DurationMinutes != 60 {
		t.Errorf("expected DurationMinutes 60, got %d", s.DurationMinutes)
	}
	if s.ProviderID != providerID {
		t.Errorf("expected ProviderID %v, got %v", providerID, s.ProviderID)
	}
}

func TestService_JSONMarshal_ProviderIDRoundtrip(t *testing.T) {
	original := domain.Service{
		ID:              uuid.New(),
		ProviderID:      uuid.New(),
		Name:            "Manicure",
		PriceCents:      2000,
		DurationMinutes: 45,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored domain.Service
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if restored.ProviderID != original.ProviderID {
		t.Errorf("ProviderID mismatch after roundtrip: got %v, want %v", restored.ProviderID, original.ProviderID)
	}
	if restored.PriceCents != original.PriceCents {
		t.Errorf("PriceCents mismatch after roundtrip: got %d, want %d", restored.PriceCents, original.PriceCents)
	}
}
