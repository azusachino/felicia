package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("FELICIA_TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("FELICIA_TEST_DATABASE_DSN is not set; skipping PostgreSQL integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open postgres test database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres test database: %v", err)
	}

	var version int64
	if err := pool.QueryRow(ctx, "SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version").Scan(&version); err != nil {
		t.Fatalf("check PostgreSQL migrations: %v (run make migrate against FELICIA_TEST_DATABASE_DSN)", err)
	}
	if version < 7 {
		t.Fatalf("PostgreSQL migrations are incomplete: got version %d, want at least 7", version)
	}
	return pool
}
