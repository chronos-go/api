package e2e

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// ── 400 Bad Request ──────────────────────────────────────────────────────────

func TestBadRequest_InvalidJSON_Provider(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	req, _ := http.NewRequest(http.MethodPost, ts.srv.URL+"/providers", strings.NewReader("not-json{{{"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	mustStatus(t, resp, http.StatusBadRequest)
	body := readBody(t, resp)
	if _, ok := body["error"]; !ok {
		t.Error("expected error field in 400 response")
	}
}

func TestBadRequest_InvalidJSON_Client(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	req, _ := http.NewRequest(http.MethodPost, ts.srv.URL+"/clients", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	mustStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestBadRequest_MissingFields_Provider(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	cases := []struct {
		name string
		body map[string]string
	}{
		{"missing name", map[string]string{
			"email": "a@b.com",
			"document": "123",
			"password": "pass1234",
		}},
		{"missing email", map[string]string{
			"name": "N",
			"document": "123",
			"password": "pass1234",
		}},
		{"missing document", map[string]string{
			"name": "N",
			"email": "a@b.com",
			"password": "pass1234",
		}},
		{"missing password", map[string]string{
			"name": "N",
			"email": "a@b.com",
			"document": "123",
		}},
		{"invalid email format", map[string]string{
			"name": "N",
			"email": "notanemail",
			"document": "123",
			"password": "pass1234",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := do(t, ts, http.MethodPost, "/providers", tc.body, "")
			mustStatus(t, resp, http.StatusBadRequest)
			resp.Body.Close()
		})
	}
}

func TestBadRequest_MissingFields_Client(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	cases := []struct {
		name string
		body map[string]string
	}{
		{"missing name", map[string]string{
			"email": "a@b.com",
			"birth_date": "1990-01-01",
			"password": "pass1234",
		}},
		{"missing email", map[string]string{
			"name": "Na",
			"birth_date": "1990-01-01",
			"password": "pass1234",
		}},
		{"missing birth_date", map[string]string{
			"name": "Na",
			"email": "a@b.com",
			"password": "pass1234",
		}},
		{"missing password", map[string]string{
			"name": "Na",
			"email": "a@b.com",
			"birth_date": "1990-01-01",
		}},
		{"name too short", map[string]string{
			"name": "X",
			"email": "a@b.com",
			"birth_date": "1990-01-01",
			"password": "pass1234",
		}},
		{"password too short", map[string]string{
			"name": "Na",
			"email": "a@b.com",
			"birth_date": "1990-01-01",
			"password": "short",
	}},
		{"invalid birth_date", map[string]string{
			"name": "Na",
			"email": "a@b.com",
			"birth_date": "not-a-date",
			"password": "pass1234",
	}},
}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := do(t, ts, http.MethodPost, "/clients", tc.body, "")
			mustStatus(t, resp, http.StatusBadRequest)
			resp.Body.Close()
		})
	}
}

func TestBadRequest_MissingFields_Service(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, token, _ := mustRegisterProvider(t, ts, uniqueEmail("svc-bad-req"), "password123")

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing name", map[string]any{"price_cents": 100, "duration_minutes": 30}},
		{"zero duration", map[string]any{"name": "S", "price_cents": 100, "duration_minutes": 0}},
		{"negative duration", map[string]any{"name": "S", "price_cents": 100, "duration_minutes": -1}},
		{"negative price", map[string]any{"name": "S", "price_cents": -1, "duration_minutes": 30}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := do(t, ts, http.MethodPost, "/services", tc.body, token)
			mustStatus(t, resp, http.StatusBadRequest)
			resp.Body.Close()
		})
	}
}

func TestBadRequest_Login_MissingFields(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	cases := []struct {
		name string
		body map[string]string
	}{
		{"missing email", map[string]string{"password": "pass", "role": "provider"}},
		{"missing password", map[string]string{"email": "a@b.com", "role": "provider"}},
		{"missing role", map[string]string{"email": "a@b.com", "password": "pass"}},
		{"invalid role", map[string]string{"email": "a@b.com", "password": "pass", "role": "admin"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := do(t, ts, http.MethodPost, "/auth/login", tc.body, "")
			mustStatus(t, resp, http.StatusBadRequest)
			resp.Body.Close()
		})
	}
}

// ── 401 Unauthorized ─────────────────────────────────────────────────────────

func TestUnauthorized_NoToken(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/providers/me"},
		{http.MethodGet, "/clients/me"},
		{http.MethodGet, "/services"},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			resp := do(t, ts, tc.method, tc.path, nil, "")
			mustStatus(t, resp, http.StatusUnauthorized)
			body := readBody(t, resp)
			if _, ok := body["error"]; !ok {
				t.Error("expected error field in 401 response")
			}
		})
	}
}

func TestUnauthorized_InvalidToken(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	resp := do(t, ts, http.MethodGet, "/providers/me", nil, "eyJhbGciOiJIUzI1NiJ9.bad.payload")
	mustStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestUnauthorized_WrongCredentials_Provider(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("wrong-creds-p")
	resp := do(t, ts, http.MethodPost, "/providers", map[string]string{
		"name": "P", "email": email, "document": "123", "password": "correct-password",
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, "wrong-password", "provider"), "")
	mustStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestUnauthorized_WrongCredentials_Client(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("wrong-creds-c")
	resp := do(t, ts, http.MethodPost, "/clients", map[string]string{
		"name": "Cl", "email": email, "birth_date": "1990-01-01", "password": "correct-password",
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, "wrong-password", "client"), "")
	mustStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestUnauthorized_NonExistentEmail(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	resp := do(t, ts, http.MethodPost, "/auth/login", loginPayload(uniqueEmail("ghost"), "anypass", "provider"), "")
	mustStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

// ── 403 Forbidden ────────────────────────────────────────────────────────────

func TestForbidden_ClientAccessingProviderRoute(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, clientToken, _ := mustRegisterClient(t, ts, uniqueEmail("forbidden-client"), "password123")

	resp := do(t, ts, http.MethodGet, "/providers/me", nil, clientToken)
	mustStatus(t, resp, http.StatusForbidden)
	body := readBody(t, resp)
	if _, ok := body["error"]; !ok {
		t.Error("expected error field in 403 response")
	}
}

func TestForbidden_ProviderAccessingClientRoute(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, providerToken, _ := mustRegisterProvider(t, ts, uniqueEmail("forbidden-provider"), "password123")

	resp := do(t, ts, http.MethodGet, "/clients/me", nil, providerToken)
	mustStatus(t, resp, http.StatusForbidden)
	resp.Body.Close()
}

// ── 404 Not Found ─────────────────────────────────────────────────────────────

func TestNotFound_Provider(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, token, _ := mustRegisterProvider(t, ts, uniqueEmail("notfound-p"), "password123")

	nonExistent := "00000000-0000-0000-0000-000000000001"
	resp := do(t, ts, http.MethodGet, "/providers/"+nonExistent, nil, token)
	mustStatus(t, resp, http.StatusNotFound)
	body := readBody(t, resp)
	if _, ok := body["error"]; !ok {
		t.Error("expected error field in 404 response")
	}
}

func TestNotFound_Service(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, token, _ := mustRegisterProvider(t, ts, uniqueEmail("notfound-s"), "password123")

	nonExistent := "00000000-0000-0000-0000-000000000002"
	resp := do(t, ts, http.MethodGet, "/services/"+nonExistent, nil, token)
	mustStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestNotFound_ProviderDetails(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, token, _ := mustRegisterProvider(t, ts, uniqueEmail("notfound-details"), "password123")

	nonExistent := "00000000-0000-0000-0000-000000000003"
	resp := do(t, ts, http.MethodGet, "/providers/"+nonExistent+"/details", nil, token)
	mustStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// ── 409 Conflict ──────────────────────────────────────────────────────────────

func TestConflict_DuplicateEmail_Provider(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("dup-provider")
	payload := map[string]string{
		"name":     "Dup",
		"email":    email,
		"document": "11122233300",
		"password": "password123",
	}

	resp := do(t, ts, http.MethodPost, "/providers", payload, "")
	mustStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Second registration with same email.
	resp = do(t, ts, http.MethodPost, "/providers", payload, "")
	mustStatus(t, resp, http.StatusConflict)
	body := readBody(t, resp)
	if _, ok := body["error"]; !ok {
		t.Error("expected error field in 409 response")
	}
}

func TestConflict_DuplicateEmail_Client(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("dup-client")
	payload := map[string]string{
		"name":       "Dup",
		"email":      email,
		"birth_date": "1990-01-01",
		"password":   "password123",
	}

	resp := do(t, ts, http.MethodPost, "/clients", payload, "")
	mustStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = do(t, ts, http.MethodPost, "/clients", payload, "")
	mustStatus(t, resp, http.StatusConflict)
	resp.Body.Close()
}

// ── 429 Too Many Requests ─────────────────────────────────────────────────────

// TestRateLimit verifies that the auth endpoint returns 429 after exceeding the
// configured limit. The server is created with limit=2 so the 3rd request fails.
func TestRateLimit(t *testing.T) {
	ts := newTestServerWithLimiter(t, 2, 60*time.Second)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("ratelimit")
	payload := loginPayload(email, "does-not-matter", "provider")

	// Requests 1 and 2 — under the limit, may return any status (401 expected
	// since the provider doesn't exist, but that's fine — we only care about
	// the rate-limit response on request 3).
	resp1 := do(t, ts, http.MethodPost, "/auth/login", payload, "")
	resp1.Body.Close()

	resp2 := do(t, ts, http.MethodPost, "/auth/login", payload, "")
	resp2.Body.Close()

	// Request 3 — must be rate-limited.
	resp3 := do(t, ts, http.MethodPost, "/auth/login", payload, "")
	mustStatus(t, resp3, http.StatusTooManyRequests)
	body := readBody(t, resp3)
	if _, ok := body["error"]; !ok {
		t.Error("expected error field in 429 response")
	}

	// Validate rate-limit headers are present.
	if resp3.Header.Get("X-RateLimit-Limit") == "" {
		t.Error("X-RateLimit-Limit header missing on 429")
	}
	if resp3.Header.Get("Retry-After") == "" {
		t.Error("Retry-After header missing on 429")
	}
}

// ── Password/hash never in responses ─────────────────────────────────────────

func TestNoPasswordInResponse_Provider(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("no-pw-provider")
	password := "password123"

	// Registration.
	resp := do(t, ts, http.MethodPost, "/providers", map[string]string{
		"name": "NoPW", "email": email, "document": "123456789", "password": password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	body := readBody(t, resp)
	assertNoField(t, body, "password")
	assertNoField(t, body, "password_hash")
	assertNoField(t, body, "hash")

	// Login.
	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "provider"), "")
	mustStatus(t, resp, http.StatusOK)
	loginBody := readBody(t, resp)
	assertNoField(t, loginBody, "password")
	assertNoField(t, loginBody, "password_hash")

	userObj, _ := loginBody["user"].(map[string]any)
	assertNoField(t, userObj, "password")

	// GET /providers/me.
	accessToken := mustField(t, loginBody, "access_token")
	resp = do(t, ts, http.MethodGet, "/providers/me", nil, accessToken)
	mustStatus(t, resp, http.StatusOK)
	meBody := readBody(t, resp)
	assertNoField(t, meBody, "password")
	assertNoField(t, meBody, "password_hash")
}

func TestNoPasswordInResponse_Client(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("no-pw-client")
	password := "password123"

	resp := do(t, ts, http.MethodPost, "/clients", map[string]string{
		"name": "NoPW", "email": email, "birth_date": "1990-01-01", "password": password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	body := readBody(t, resp)
	assertNoField(t, body, "password")
	assertNoField(t, body, "password_hash")

	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "client"), "")
	mustStatus(t, resp, http.StatusOK)
	loginBody := readBody(t, resp)
	assertNoField(t, loginBody, "password")

	accessToken := mustField(t, loginBody, "access_token")
	resp = do(t, ts, http.MethodGet, "/clients/me", nil, accessToken)
	mustStatus(t, resp, http.StatusOK)
	meBody := readBody(t, resp)
	assertNoField(t, meBody, "password")
}

// TestNoTokenHashInSessions verifies that token_hash never leaks to callers.
func TestNoTokenHashInSessions(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("no-hash")
	password := "password123"

	resp := do(t, ts, http.MethodPost, "/providers", map[string]string{
		"name": "NH", "email": email, "document": "000", "password": password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "provider"), "")
	mustStatus(t, resp, http.StatusOK)
	body := readBody(t, resp)
	assertNoField(t, body, "token_hash")

	refreshToken := mustField(t, body, "refresh_token")
	resp = do(t, ts, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	mustStatus(t, resp, http.StatusOK)
	refreshBody := readBody(t, resp)
	assertNoField(t, refreshBody, "token_hash")
}

// ── Security headers ──────────────────────────────────────────────────────────

func TestSecurityHeaders(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	resp, err := http.Get(ts.srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	}

	for header, expected := range headers {
		got := resp.Header.Get(header)
		if got != expected {
			t.Errorf("header %q: expected %q, got %q", header, expected, got)
		}
	}
}

// TestCacheControlNoStore_Login verifies Cache-Control: no-store on login.
func TestCacheControlNoStore_Login(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("cc-login")
	password := "password123"

	resp := do(t, ts, http.MethodPost, "/providers", map[string]string{
		"name": "CC", "email": email, "document": "000", "password": password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "provider"), "")
	mustStatus(t, resp, http.StatusOK)
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Login: expected Cache-Control: no-store, got %q", cc)
	}
	resp.Body.Close()
}

// TestCacheControlNoStore_Refresh verifies Cache-Control: no-store on refresh.
func TestCacheControlNoStore_Refresh(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	email := uniqueEmail("cc-refresh")
	password := "password123"

	resp := do(t, ts, http.MethodPost, "/providers", map[string]string{
		"name": "CC", "email": email, "document": "000", "password": password,
	}, "")
	mustStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = do(t, ts, http.MethodPost, "/auth/login", loginPayload(email, password, "provider"), "")
	mustStatus(t, resp, http.StatusOK)
	loginBody := readBody(t, resp)
	refreshToken := mustField(t, loginBody, "refresh_token")

	resp = do(t, ts, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	mustStatus(t, resp, http.StatusOK)
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Refresh: expected Cache-Control: no-store, got %q", cc)
	}
	resp.Body.Close()
}

// ── Miscellaneous edge cases ──────────────────────────────────────────────────

// TestBadRequest_InvalidUUID verifies that routes with UUID params return 400
// when a non-UUID string is provided.
func TestBadRequest_InvalidUUID(t *testing.T) {
	ts := newTestServer(t)
	cleanupTables(t, ts.pool)

	_, token, _ := mustRegisterProvider(t, ts, uniqueEmail("invalid-uuid"), "password123")

	cases := []struct {
		path string
	}{
		{"/providers/not-a-uuid"},
		{"/providers/not-a-uuid/details"},
		{"/services/not-a-uuid"},
	}

	for _, tc := range cases {
		t.Run("GET "+tc.path, func(t *testing.T) {
			resp := do(t, ts, http.MethodGet, tc.path, nil, token)
			mustStatus(t, resp, http.StatusBadRequest)
			resp.Body.Close()
		})
	}
}
