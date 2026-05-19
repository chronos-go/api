package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Connect abre e valida a conexão com o PostgreSQL usando DATABASE_URL.
// A aplicação encerra com erro claro se o banco estiver indisponível.
func Connect() *sql.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("database is unreachable: %v\nDATABASE_URL=%s", err, dsn)
	}

	fmt.Println("database connection established")
	return db
}
