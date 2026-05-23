package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func OpenTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("COMMUNE_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://commune:commune@localhost:5432/commune?sslmode=disable"
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	config.MaxConns = 1
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Skipf("test database unavailable: %v", err)
	}
	if _, err := pool.Exec(ctx, "SELECT pg_advisory_lock(2026052201)"); err != nil {
		pool.Close()
		t.Fatalf("acquire test database lock: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
