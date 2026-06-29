package e2e

import (
	"net/http"
	"testing"
)

// TestOwnership_ProviderBCannotModifyProviderAService verifies that a provider
// cannot modify or delete another provider's service (expected 403).
func TestOwnership_ProviderBCannotModifyProviderAService(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	// Register provider A and create a service.
	_, tokenA, _ := mustRegisterProvider(t, ts, uniqueEmail("owner-a"), "passwordA123")

	resp := do(t, ts, http.MethodPost, "/services", map[string]any{
		"name":             "Service A",
		"description":      "Belongs to A",
		"price_cents":      2000,
		"duration_minutes": 30,
	}, tokenA)
	mustStatus(t, resp, http.StatusCreated)
	svcBody := readBody(t, resp)
	serviceAID := mustField(t, svcBody, "id")

	// Register provider B.
	_, tokenB, _ := mustRegisterProvider(t, ts, uniqueEmail("owner-b"), "passwordB123")

	// B tries to update A's service — must get 403.
	t.Run("PUT service of provider A by provider B", func(t *testing.T) {
		resp := do(t, ts, http.MethodPut, "/services/"+serviceAID, map[string]any{
			"name":             "Hijacked Service",
			"description":      "Provider B takeover",
			"price_cents":      9999,
			"duration_minutes": 120,
		}, tokenB)
		mustStatus(t, resp, http.StatusForbidden)
		resp.Body.Close()
	})

	// B tries to delete A's service — must get 403.
	t.Run("DELETE service of provider A by provider B", func(t *testing.T) {
		resp := do(t, ts, http.MethodDelete, "/services/"+serviceAID, nil, tokenB)
		mustStatus(t, resp, http.StatusForbidden)
		resp.Body.Close()
	})

	// Confirm A's service is untouched.
	t.Run("Service A still exists and is unchanged", func(t *testing.T) {
		resp := do(t, ts, http.MethodGet, "/services/"+serviceAID, nil, tokenA)
		mustStatus(t, resp, http.StatusOK)
		body := readBody(t, resp)
		if body["name"] != "Service A" {
			t.Errorf("expected name 'Service A', got %v", body["name"])
		}
	})
}

// TestOwnership_ProviderCanModifyOwnService confirms the positive case:
// a provider CAN modify their own service.
func TestOwnership_ProviderCanModifyOwnService(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, tokenA, _ := mustRegisterProvider(t, ts, uniqueEmail("self-owner"), "password123")

	resp := do(t, ts, http.MethodPost, "/services", map[string]any{
		"name":             "My Service",
		"price_cents":      1000,
		"duration_minutes": 15,
	}, tokenA)
	mustStatus(t, resp, http.StatusCreated)
	svcBody := readBody(t, resp)
	serviceID := mustField(t, svcBody, "id")

	// Update own service — must succeed.
	resp = do(t, ts, http.MethodPut, "/services/"+serviceID, map[string]any{
		"name":             "My Updated Service",
		"price_cents":      1500,
		"duration_minutes": 20,
	}, tokenA)
	mustStatus(t, resp, http.StatusOK)
	updated := readBody(t, resp)
	if updated["name"] != "My Updated Service" {
		t.Errorf("expected updated name, got %v", updated["name"])
	}

	// Delete own service — must succeed.
	resp = do(t, ts, http.MethodDelete, "/services/"+serviceID, nil, tokenA)
	mustStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()
}
