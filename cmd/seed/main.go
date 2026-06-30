// Command seed populates the database with reproducible demonstration data
// for the MVP. It reads DATABASE_URL from the environment, creates tables if
// they do not exist, and inserts deterministic seed records.
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/seed
//
// To clean up seed data:
//
//	DATABASE_URL=postgres://... go run ./cmd/seed -clean
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/chronos-go/api/internal/database"
	"github.com/chronos-go/api/internal/seed"
)

func main() {
	clean := flag.Bool("clean", false, "remove seed data instead of inserting")
	flag.Parse()

	pool := database.Connect()
	defer pool.Close()

	if *clean {
		if err := seed.Cleanup(pool); err != nil {
			log.Fatalf("seed cleanup failed: %v", err)
		}
		return
	}

	// Ensure tables exist (idempotent — safe if migrations already applied).
	if err := seed.EnsureTables(pool); err != nil {
		log.Fatalf("ensuring tables: %v", err)
	}

	if err := seed.Run(pool); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	fmt.Println("\nseed completed successfully")
}