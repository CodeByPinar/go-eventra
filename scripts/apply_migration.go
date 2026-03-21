package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	filePath := flag.String("file", "", "Path to SQL migration file")
	dbURL := flag.String("db-url", "", "PostgreSQL connection string")
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "-file is required")
		os.Exit(1)
	}

	resolvedDBURL := *dbURL
	if resolvedDBURL == "" {
		resolvedDBURL = os.Getenv("DB_URL")
	}
	if resolvedDBURL == "" {
		resolvedDBURL = "postgres://postgres:postgres@localhost:5432/eventra?sslmode=disable"
	}

	sqlBytes, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read migration file: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, resolvedDBURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if _, err = pool.Exec(ctx, string(sqlBytes)); err != nil {
		fmt.Fprintf(os.Stderr, "apply migration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("migration applied successfully: %s\n", *filePath)
}
