// Package seed provides a reproducible mechanism to populate the database
// with demonstration data for the MVP. All UUIDs are deterministic (UUIDv5
// based on email), passwords are bcrypt-hashed, and every operation is
// idempotent — running the seed multiple times produces the same result.
package seed

import (
	"context"
	"fmt"

	"github.com/chronos-go/api/internal/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Deterministic UUID helpers ──────────────────────────────────────────────

var namespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace

func deterministicUUID(email string) uuid.UUID {
	return uuid.NewSHA1(namespace, []byte(email))
}

// ─── Seed data ───────────────────────────────────────────────────────────────

// DemoPassword is the plain-text password used for all seed accounts.
// It is intentionally simple so that reviewers and testers can log in easily.
const DemoPassword = "demo123456"

type seedClient struct {
	ID        uuid.UUID
	Name      string
	Email     string
	BirthDate string // YYYY-MM-DD
	Password  string // bcrypt hash, set at runtime
}

type seedProvider struct {
	ID       uuid.UUID
	Name     string
	Email    string
	Document string
	Password string // bcrypt hash, set at runtime
	Services []seedService
}

type seedService struct {
	Name            string
	Description     string
	PriceCents      int
	DurationMinutes int
}

// demoClient returns the single demonstration client.
func demoClient() seedClient {
	return seedClient{
		ID:        deterministicUUID("client-demo@chronos.app"),
		Name:      "Cliente Demonstração",
		Email:     "client-demo@chronos.app",
		BirthDate: "1990-05-15",
	}
}

// demoProviders returns the two demonstration providers with their services.
func demoProviders() []seedProvider {
	return []seedProvider{
		{
			ID:       deterministicUUID("vintage@chronos.app"),
			Name:     "Barbearia Vintage",
			Email:    "vintage@chronos.app",
			Document: "12345678000199",
			Services: []seedService{
				{Name: "Corte Masculino", Description: "Corte com tesoura e máquina", PriceCents: 3500, DurationMinutes: 40},
				{Name: "Barba Completa", Description: "Aparação e modelagem de barba", PriceCents: 2500, DurationMinutes: 30},
				{Name: "Corte + Barba", Description: "Combo corte masculino + barba", PriceCents: 5000, DurationMinutes: 60},
			},
		},
		{
			ID:       deterministicUUID("studio@chronos.app"),
			Name:     "Studio Beleza",
			Email:    "studio@chronos.app",
			Document: "98765432000188",
			Services: []seedService{
				{Name: "Corte Feminino", Description: "Corte personalizado feminino", PriceCents: 6000, DurationMinutes: 60},
				{Name: "Escova", Description: "Escova modeladora", PriceCents: 4000, DurationMinutes: 45},
				{Name: "Coloração", Description: "Coloração completa", PriceCents: 12000, DurationMinutes: 120},
			},
		},
	}
}

// ─── Run ─────────────────────────────────────────────────────────────────────

// Run inserts seed data into the database connected via pool.
// It is idempotent: existing records are updated, never duplicated.
// Only IDs, emails and demo instructions are printed to stdout.
func Run(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// Hash the demo password once and reuse it.
	hashedPassword, err := crypto.Hash(DemoPassword)
	if err != nil {
		return fmt.Errorf("hashing demo password: %w", err)
	}

	// ── Client ────────────────────────────────────────────────────────────
	client := demoClient()
	if err := upsertClient(ctx, pool, client, hashedPassword); err != nil {
		return fmt.Errorf("seeding client: %w", err)
	}
	fmt.Printf("client: id=%s  email=%s\n", client.ID, client.Email)

	// ── Providers + Services ──────────────────────────────────────────────
	for _, p := range demoProviders() {
		if err := upsertProvider(ctx, pool, p, hashedPassword); err != nil {
			return fmt.Errorf("seeding provider %q: %w", p.Email, err)
		}
		fmt.Printf("provider: id=%s  email=%s  name=%q\n", p.ID, p.Email, p.Name)

		for _, svc := range p.Services {
			svcID := deterministicUUID(p.Email + ":" + svc.Name)
			if err := upsertService(ctx, pool, svcID, p.ID, svc); err != nil {
				return fmt.Errorf("seeding service %q for provider %q: %w", svc.Name, p.Email, err)
			}
			fmt.Printf("  service: id=%s  name=%q  price=R$%.2f\n", svcID, svc.Name, float64(svc.PriceCents)/100)
		}
	}

	fmt.Println()
	fmt.Println("── Demo credentials ──────────────────────────────────────")
	fmt.Printf("  Client:  %s / %s\n", client.Email, DemoPassword)
	for _, p := range demoProviders() {
		fmt.Printf("  Provider: %s / %s\n", p.Email, DemoPassword)
	}
	fmt.Println("───────────────────────────────────────────────────────────")

	return nil
}

// ─── Upsert helpers ──────────────────────────────────────────────────────────

func upsertClient(ctx context.Context, pool *pgxpool.Pool, c seedClient, hashedPassword string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO clients (id, name, email, birth_date, password, created_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (email) DO UPDATE
			SET name = EXCLUDED.name,
			    birth_date = EXCLUDED.birth_date,
			    password = EXCLUDED.password
	`, c.ID, c.Name, c.Email, c.BirthDate, hashedPassword)
	return err
}

func upsertProvider(ctx context.Context, pool *pgxpool.Pool, p seedProvider, hashedPassword string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO providers (id, name, email, document, password, created_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (email) DO UPDATE
			SET name = EXCLUDED.name,
			    document = EXCLUDED.document,
			    password = EXCLUDED.password
	`, p.ID, p.Name, p.Email, p.Document, hashedPassword)
	return err
}

func upsertService(ctx context.Context, pool *pgxpool.Pool, id, providerID uuid.UUID, svc seedService) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO services (id, provider_id, name, description, price_cents, duration_minutes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (id) DO NOTHING
	`, id, providerID, svc.Name, svc.Description, int32(svc.PriceCents), int32(svc.DurationMinutes))
	return err
}

// ─── Cleanup ─────────────────────────────────────────────────────────────────

// Cleanup removes all seed data from the database. It deletes only the records
// that were created by the seed (identified by deterministic UUIDs).
func Cleanup(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// Collect all deterministic IDs.
	ids := []uuid.UUID{demoClient().ID}
	for _, p := range demoProviders() {
		ids = append(ids, p.ID)
		for _, svc := range p.Services {
			ids = append(ids, deterministicUUID(p.Email+":"+svc.Name))
		}
	}

	// Delete services first (foreign key), then providers, then client.
	for _, id := range ids {
		_, _ = pool.Exec(ctx, `DELETE FROM services WHERE id = $1`, id)
	}
	for _, p := range demoProviders() {
		_, _ = pool.Exec(ctx, `DELETE FROM providers WHERE id = $1`, p.ID)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM clients WHERE id = $1`, demoClient().ID)

	fmt.Println("seed data cleaned up")
	return nil
}

// ─── Ensure tables exist ─────────────────────────────────────────────────────

// EnsureTables creates the required tables if they do not exist.
// This is useful when running the seed against an empty database that has not
// had migrations applied yet.
func EnsureTables(pool *pgxpool.Pool) error {
	ctx := context.Background()

	_, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "pgcrypto"`)
	if err != nil {
		return fmt.Errorf("creating pgcrypto extension: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS clients (
			id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			name       TEXT        NOT NULL,
			email      TEXT        NOT NULL UNIQUE,
			birth_date DATE        NOT NULL,
			password   TEXT        NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS providers (
			id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			name       TEXT        NOT NULL,
			email      TEXT        NOT NULL UNIQUE,
			document   TEXT        NOT NULL,
			password   TEXT        NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS services (
			id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			provider_id      UUID        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
			name             TEXT        NOT NULL,
			description      TEXT        NOT NULL DEFAULT '',
			price_cents      INTEGER     NOT NULL CHECK (price_cents >= 0),
			duration_minutes INTEGER     NOT NULL CHECK (duration_minutes > 0),
			created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS services_provider_id_idx ON services(provider_id)`,
	}

	for _, q := range queries {
		if _, err := pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("creating table: %w", err)
		}
	}

	return nil
}

