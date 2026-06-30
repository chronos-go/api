package seed_test

import (
	"context"
	"os"
	"testing"

	"github.com/chronos-go/api/internal/crypto"
	"github.com/chronos-go/api/internal/seed"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── TestMain ────────────────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	if os.Getenv("DATABASE_URL") == "" {
		// Skip integration tests when no database is available.
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func connectPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func cleanupSeedData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	t.Cleanup(func() {
		_ = seed.Cleanup(pool)
	})
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestRun_InsertsClient(t *testing.T) {
	pool := connectPool(t)
	cleanupSeedData(t, pool)

	if err := seed.EnsureTables(pool); err != nil {
		t.Fatalf("EnsureTables: %v", err)
	}

	if err := seed.Run(pool); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify client was inserted.
	var count int
	err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM clients WHERE email = 'client-demo@chronos.app'`).Scan(&count)
	if err != nil {
		t.Fatalf("query client: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 client, got %d", count)
	}

	// Verify password is hashed (not plain text).
	var password string
	err = pool.QueryRow(context.Background(), `SELECT password FROM clients WHERE email = 'client-demo@chronos.app'`).Scan(&password)
	if err != nil {
		t.Fatalf("query client password: %v", err)
	}
	if password == seed.DemoPassword {
		t.Fatal("password stored in plain text — must be bcrypt hash")
	}
	if !crypto.Compare(password, seed.DemoPassword) {
		t.Fatal("stored hash does not match demo password")
	}
}

func TestRun_InsertsProviders(t *testing.T) {
	pool := connectPool(t)
	cleanupSeedData(t, pool)

	if err := seed.EnsureTables(pool); err != nil {
		t.Fatalf("EnsureTables: %v", err)
	}

	if err := seed.Run(pool); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var count int
	err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM providers`).Scan(&count)
	if err != nil {
		t.Fatalf("query providers: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 providers, got %d", count)
	}
}

func TestRun_InsertsServices(t *testing.T) {
	pool := connectPool(t)
	cleanupSeedData(t, pool)

	if err := seed.EnsureTables(pool); err != nil {
		t.Fatalf("EnsureTables: %v", err)
	}

	if err := seed.Run(pool); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var count int
	err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM services`).Scan(&count)
	if err != nil {
		t.Fatalf("query services: %v", err)
	}
	// 3 services for Barbearia Vintage + 3 for Studio Beleza
	if count != 6 {
		t.Fatalf("expected 6 services, got %d", count)
	}
}

func TestRun_Idempotent(t *testing.T) {
	pool := connectPool(t)
	cleanupSeedData(t, pool)

	if err := seed.EnsureTables(pool); err != nil {
		t.Fatalf("EnsureTables: %v", err)
	}

	// Run seed twice.
	if err := seed.Run(pool); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if err := seed.Run(pool); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	// Verify counts remain the same.
	var clientCount int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM clients`).Scan(&clientCount)
	if clientCount != 1 {
		t.Fatalf("expected 1 client after second run, got %d", clientCount)
	}

	var providerCount int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM providers`).Scan(&providerCount)
	if providerCount != 2 {
		t.Fatalf("expected 2 providers after second run, got %d", providerCount)
	}

	var serviceCount int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM services`).Scan(&serviceCount)
	if serviceCount != 6 {
		t.Fatalf("expected 6 services after second run, got %d", serviceCount)
	}
}

func TestRun_DeterministicUUIDs(t *testing.T) {
	pool := connectPool(t)
	cleanupSeedData(t, pool)

	if err := seed.EnsureTables(pool); err != nil {
		t.Fatalf("EnsureTables: %v", err)
	}

	if err := seed.Run(pool); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify that the client UUID is deterministic (SHA-1 based on email).
	expectedClientID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("client-demo@chronos.app"))
	var actualClientID uuid.UUID
	err := pool.QueryRow(context.Background(), `SELECT id FROM clients WHERE email = 'client-demo@chronos.app'`).Scan(&actualClientID)
	if err != nil {
		t.Fatalf("query client id: %v", err)
	}
	if actualClientID != expectedClientID {
		t.Fatalf("expected client id %s, got %s", expectedClientID, actualClientID)
	}
}

func TestCleanup_RemovesSeedData(t *testing.T) {
	pool := connectPool(t)

	if err := seed.EnsureTables(pool); err != nil {
		t.Fatalf("EnsureTables: %v", err)
	}

	if err := seed.Run(pool); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if err := seed.Cleanup(pool); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	var count int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM clients`).Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 clients after cleanup, got %d", count)
	}

	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM providers`).Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 providers after cleanup, got %d", count)
	}

	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM services`).Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 services after cleanup, got %d", count)
	}
}

func TestRun_OnPrepopulatedDatabase(t *testing.T) {
	pool := connectPool(t)
	cleanupSeedData(t, pool)

	if err := seed.EnsureTables(pool); err != nil {
		t.Fatalf("EnsureTables: %v", err)
	}

	// Insert a different client first.
	_, err := pool.Exec(context.Background(), `
		INSERT INTO clients (id, name, email, birth_date, password, created_at)
		VALUES ($1, 'Existing Client', 'existing@example.com', '2000-01-01', 'hash', now())
	`, uuid.New())
	if err != nil {
		t.Fatalf("insert existing client: %v", err)
	}

	// Run seed — should not affect existing records.
	if err := seed.Run(pool); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var count int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM clients`).Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2 clients (1 existing + 1 seed), got %d", count)
	}

	// Existing client should remain untouched.
	var name string
	err = pool.QueryRow(context.Background(), `SELECT name FROM clients WHERE email = 'existing@example.com'`).Scan(&name)
	if err != nil {
		t.Fatalf("query existing client: %v", err)
	}
	if name != "Existing Client" {
		t.Fatalf("expected existing client name 'Existing Client', got %q", name)
	}
}