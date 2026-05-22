package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect abre e valida a conexão com o PostgreSQL usando DATABASE_URL.
// A aplicação encerra com erro claro se o banco estiver indisponível.
func Connect() *pgxpool.Pool {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("database is unreachable: %v\nDATABASE_URL=%s", err, dsn)
	}

	fmt.Println("database connection established")
	return pool
}
