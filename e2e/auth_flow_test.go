package e2e

import (
	"net/http"
	"testing"
)

// TestTokenValidation_NoAuthHeader tests protected routes without a token.
func TestTokenValidation_NoAuthHeader(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/providers/me"},
		{http.MethodGet, "/clients/me"},
		{http.MethodGet, "/services"},
		{http.MethodPost, "/services"},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			resp := do(t, ts, tc.method, tc.path, nil, "")
			mustStatus(t, resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}

// TestTokenValidation_InvalidToken tests protected routes with a malformed token.
func TestTokenValidation_InvalidToken(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	invalidTokens := []string{
		"not-a-jwt",
		"Bearer",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
		"totally-garbage-token-string",
	}

	for _, tok := range invalidTokens {
		t.Run(tok[:min(20, len(tok))], func(t *testing.T) {
			resp := do(t, ts, http.MethodGet, "/providers/me", nil, tok)
			mustStatus(t, resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}

// TestTokenValidation_ClientTokenOnProviderRoute tests that a client token
// cannot access provider-only endpoints.
func TestTokenValidation_ClientTokenOnProviderRoute(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, clientToken, _ := mustRegisterClient(t, ts, uniqueEmail("tok-client-on-prov"), "password123")

	// Provider-only routes.
	routes := []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, "/providers/me", nil},
		{http.MethodPost, "/services", map[string]any{
			"name":             "S",
			"price_cents":      100,
			"duration_minutes": 30,
		}},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			resp := do(t, ts, tc.method, tc.path, tc.body, clientToken)
			mustStatus(t, resp, http.StatusForbidden)
			resp.Body.Close()
		})
	}
}

// TestTokenValidation_ProviderTokenOnClientRoute tests that a provider token
// cannot access client-only endpoints.
func TestTokenValidation_ProviderTokenOnClientRoute(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, providerToken, _ := mustRegisterProvider(t, ts, uniqueEmail("tok-prov-on-client"), "password123")

	// Client-only routes.
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/clients/me"},
		{http.MethodPut, "/clients/me"},
		{http.MethodDelete, "/clients/me"},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			resp := do(t, ts, tc.method, tc.path, nil, providerToken)
			mustStatus(t, resp, http.StatusForbidden)
			resp.Body.Close()
		})
	}
}

// TestRefreshToken_ReplayDetection verifies that reusing an already-rotated
// refresh token returns 401.
func TestRefreshToken_ReplayDetection(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, _, refreshToken := mustRegisterProvider(t, ts, uniqueEmail("replay"), "pass12345")

	// First refresh — valid.
	resp := do(t, ts, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	mustStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Replay the original refresh token — must fail.
	resp = do(t, ts, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	mustStatus(t, resp, http.StatusUnauthorized)
	body := readBody(t, resp)
	if _, ok := body["error"]; !ok {
		t.Error("replay response missing error field")
	}
}

// TestLogout_RevokedSessionFails verifies that a refresh token cannot be used
// after the session has been revoked via logout.
func TestLogout_RevokedSessionFails(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, _, refreshToken := mustRegisterProvider(t, ts, uniqueEmail("logout-revoke"), "pass12345")

	// Logout.
	resp := do(t, ts, http.MethodPost, "/auth/logout", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	mustStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Attempt to refresh with the now-revoked token.
	resp = do(t, ts, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	mustStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

// TestRefreshToken_Missing verifies that /auth/refresh without a body returns 400.
func TestRefreshToken_Missing(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	resp := do(t, ts, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": "",
	}, "")
	mustStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
