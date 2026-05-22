package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hanzc0106/commune/apps/api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func OpenTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("COMMUNE_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://commune:commune@localhost:5432/commune?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		t.Skipf("test database unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
