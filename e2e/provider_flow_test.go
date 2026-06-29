package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

// TestProviderFlow executes the full provider lifecycle end-to-end:
//
//  1. Register a new provider
//  2. Login and obtain access + refresh tokens
//  3. Access a protected route
//  4. Create a service (provider_id resolved from JWT sub)
//  5. List services and find the created one
//  6. Get service by ID
//  7. Update the service
//  8. Get provider details with nested services
//  9. Delete the service
//  10. Rotate the refresh token
//  11. Confirm the old refresh token fails (replay detection)
//  12. Logout
//  13. Confirm the revoked refresh token fails
func TestProviderFlow(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("provider-flow")
	password := "superSecure123"

	// ── 1. Register provider ─────────────────────────────────────────────────
	resp := do(t, ts, http.MethodPost, "/providers", map[string]string{
		"name":     "Flow Provider",
		"email":    email,
		"document": "12345678000199",
		"password": password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	body := readBody(t, resp)
	providerID := mustField(t, body, "id")
	assertNoField(t, body, "password")
	assertNoField(t, body, "password_hash")

	if body["email"] != email {
		t.Errorf("expected email %q, got %v", email, body["email"])
	}

	// ── 2. Login ──────────────────────────────────────────────────────────────
	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "provider"), "")
	mustStatus(t, resp, http.StatusOK)

	// Validate Cache-Control: no-store
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("expected Cache-Control: no-store on login, got %q", cc)
	}

	loginBody := readBody(t, resp)
	accessToken := mustField(t, loginBody, "access_token")
	refreshToken := mustField(t, loginBody, "refresh_token")
	mustField(t, loginBody, "expires_at")
	mustField(t, loginBody, "refresh_expires_at")
	assertNoField(t, loginBody, "password")

	userObj, ok := loginBody["user"].(map[string]any)
	if !ok {
		t.Fatal("login response missing user object")
	}
	if userObj["role"] != "provider" {
		t.Errorf("expected role=provider, got %v", userObj["role"])
	}
	assertNoField(t, userObj, "password")

	// ── 3. Access protected route ─────────────────────────────────────────────
	resp = do(t, ts, http.MethodGet, "/providers/me", nil, accessToken)
	mustStatus(t, resp, http.StatusOK)
	meBody := readBody(t, resp)
	assertNoField(t, meBody, "password")
	if meBody["id"] != providerID {
		t.Errorf("providers/me: expected id %q, got %v", providerID, meBody["id"])
	}

	// ── 4. Create service (provider_id from JWT sub) ──────────────────────────
	resp = do(t, ts, http.MethodPost, "/services", map[string]any{
		"name":             "Haircut",
		"description":      "Premium haircut service",
		"price_cents":      3500,
		"duration_minutes": 45,
	}, accessToken)
	mustStatus(t, resp, http.StatusCreated)
	svcBody := readBody(t, resp)
	serviceID := mustField(t, svcBody, "id")
	if svcBody["provider_id"] != providerID {
		t.Errorf("service provider_id: expected %q, got %v", providerID, svcBody["provider_id"])
	}

	// ── 5. List services ──────────────────────────────────────────────────────
	resp = do(t, ts, http.MethodGet, "/services", nil, accessToken)
	mustStatus(t, resp, http.StatusOK)
	var listBody []any
	func() {
		defer resp.Body.Close()
		if err := decodeArray(t, resp, &listBody); err != nil {
			t.Fatalf("list services: decode: %v", err)
		}
	}()

	found := false
	for _, item := range listBody {
		svc, _ := item.(map[string]any)
		if svc["id"] == serviceID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("created service %q not found in list", serviceID)
	}

	// ── 6. Get service by ID ──────────────────────────────────────────────────
	resp = do(t, ts, http.MethodGet, "/services/"+serviceID, nil, accessToken)
	mustStatus(t, resp, http.StatusOK)
	getBody := readBody(t, resp)
	if getBody["id"] != serviceID {
		t.Errorf("GetByID: expected id %q, got %v", serviceID, getBody["id"])
	}

	// ── 7. Update service ─────────────────────────────────────────────────────
	resp = do(t, ts, http.MethodPut, "/services/"+serviceID, map[string]any{
		"name":             "Premium Haircut",
		"description":      "Updated description",
		"price_cents":      4000,
		"duration_minutes": 60,
	}, accessToken)
	mustStatus(t, resp, http.StatusOK)
	updateBody := readBody(t, resp)
	if updateBody["name"] != "Premium Haircut" {
		t.Errorf("update: expected name 'Premium Haircut', got %v", updateBody["name"])
	}
	if updateBody["price_cents"] != float64(4000) {
		t.Errorf("update: expected price_cents 4000, got %v", updateBody["price_cents"])
	}

	// ── 8. Provider details with nested services ──────────────────────────────
	resp = do(t, ts, http.MethodGet, fmt.Sprintf("/providers/%s/details", providerID), nil, accessToken)
	mustStatus(t, resp, http.StatusOK)
	detailsBody := readBody(t, resp)
	assertNoField(t, detailsBody, "password")

	services, ok := detailsBody["services"].([]any)
	if !ok {
		t.Fatal("provider details: services field is missing or not an array")
	}
	foundInDetails := false
	for _, item := range services {
		svc, _ := item.(map[string]any)
		if svc["id"] == serviceID {
			foundInDetails = true
			break
		}
	}
	if !foundInDetails {
		t.Errorf("service %q not found in provider details", serviceID)
	}

	// ── 9. Delete service ─────────────────────────────────────────────────────
	resp = do(t, ts, http.MethodDelete, "/services/"+serviceID, nil, accessToken)
	mustStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Confirm 404 after deletion
	resp = do(t, ts, http.MethodGet, "/services/"+serviceID, nil, accessToken)
	mustStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()

	// ── 10. Rotate refresh token ──────────────────────────────────────────────
	resp = do(t, ts, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	mustStatus(t, resp, http.StatusOK)

	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("expected Cache-Control: no-store on refresh, got %q", cc)
	}
	refreshBody := readBody(t, resp)
	newRefreshToken := mustField(t, refreshBody, "refresh_token")
	mustField(t, refreshBody, "access_token")

	// ── 11. Old refresh token must fail (replay detection) ────────────────────
	resp = do(t, ts, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": refreshToken, // original, already rotated
	}, "")
	mustStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()

	// ── 12. Logout ────────────────────────────────────────────────────────────
	resp = do(t, ts, http.MethodPost, "/auth/logout", map[string]string{
		"refresh_token": newRefreshToken,
	}, "")
	mustStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// ── 13. Revoked refresh token must fail ───────────────────────────────────
	resp = do(t, ts, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": newRefreshToken,
	}, "")
	mustStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}
