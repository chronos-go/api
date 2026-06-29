package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestClientFlow executes the full client lifecycle end-to-end:
//
//  1. Register a new client
//  2. Login and validate tokens
//  3. GET /clients/me — authenticated self-lookup
//  4. PUT /clients/me — update name
//  5. DELETE /clients/me — self-deletion
func TestClientFlow(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("client-flow")
	password := "clientPass456"

	// ── 1. Register client ────────────────────────────────────────────────────
	resp := do(t, ts, http.MethodPost, "/clients", map[string]string{
		"name":       "Test Client",
		"email":      email,
		"birth_date": "1995-06-20",
		"password":   password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	body := readBody(t, resp)
	clientID := mustField(t, body, "id")
	assertNoField(t, body, "password")
	assertNoField(t, body, "password_hash")

	if body["email"] != email {
		t.Errorf("expected email %q, got %v", email, body["email"])
	}
	if body["birth_date"] != "1995-06-20" {
		t.Errorf("expected birth_date '1995-06-20', got %v", body["birth_date"])
	}

	// ── 2. Login ──────────────────────────────────────────────────────────────
	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "client"), "")
	mustStatus(t, resp, http.StatusOK)

	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("expected Cache-Control: no-store on login, got %q", cc)
	}

	loginBody := readBody(t, resp)
	accessToken := mustField(t, loginBody, "access_token")
	mustField(t, loginBody, "refresh_token")
	assertNoField(t, loginBody, "password")

	userObj, ok := loginBody["user"].(map[string]any)
	if !ok {
		t.Fatal("login response missing user object")
	}
	if userObj["role"] != "client" {
		t.Errorf("expected role=client, got %v", userObj["role"])
	}
	if userObj["id"] != clientID {
		t.Errorf("login user id mismatch: expected %q, got %v", clientID, userObj["id"])
	}

	// ── 3. GET /clients/me ────────────────────────────────────────────────────
	resp = do(t, ts, http.MethodGet, "/clients/me", nil, accessToken)
	mustStatus(t, resp, http.StatusOK)
	meBody := readBody(t, resp)
	assertNoField(t, meBody, "password")
	if meBody["id"] != clientID {
		t.Errorf("clients/me: expected id %q, got %v", clientID, meBody["id"])
	}

	// ── 4. PUT /clients/me ────────────────────────────────────────────────────
	resp = do(t, ts, http.MethodPut, "/clients/me", map[string]string{
		"name":       "Updated Client",
		"email":      email,
		"birth_date": "1995-06-20",
	}, accessToken)
	mustStatus(t, resp, http.StatusOK)
	updateBody := readBody(t, resp)
	assertNoField(t, updateBody, "password")
	if updateBody["name"] != "Updated Client" {
		t.Errorf("PUT clients/me: expected name 'Updated Client', got %v", updateBody["name"])
	}

	// ── 5. DELETE /clients/me ─────────────────────────────────────────────────
	resp = do(t, ts, http.MethodDelete, "/clients/me", nil, accessToken)
	mustStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// After deletion, GET /clients/me should return 404.
	resp = do(t, ts, http.MethodGet, "/clients/me", nil, accessToken)
	mustStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// TestClientLoginWithWrongRole confirms that a client cannot login with role=provider.
func TestClientLoginWithWrongRole(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("client-wrong-role")
	password := "clientPass456"

	resp := do(t, ts, http.MethodPost, "/clients", map[string]string{
		"name":       "Wrong Role Client",
		"email":      email,
		"birth_date": "1990-03-10",
		"password":   password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Login with role=provider for a client email should return 401.
	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "provider"), "")
	mustStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

// TestProviderLoginWithWrongRole confirms that a provider cannot login with role=client.
func TestProviderLoginWithWrongRole(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("provider-wrong-role")
	password := "providerPass789"

	resp := do(t, ts, http.MethodPost, "/providers", map[string]string{
		"name":     "Wrong Role Provider",
		"email":    email,
		"document": "98765432000100",
		"password": password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Login with role=client for a provider email should return 401.
	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "client"), "")
	mustStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

// decodeArray is a helper used in other test files to decode JSON arrays from
// an http.Response. It is defined here so that it lives next to the other
// helpers; the caller is responsible for closing resp.Body.
func decodeArray(t *testing.T, resp *http.Response, out any) error {
	t.Helper()
	return json.NewDecoder(resp.Body).Decode(out)
}
