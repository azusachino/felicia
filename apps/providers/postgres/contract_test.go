package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/azusachino/felicia/apps/providers/contract"
	"github.com/azusachino/felicia/apps/providers/postgres"
)

func TestRepositoryContract(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN environment variable not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, "TRUNCATE TABLE tb_journal CASCADE"); err != nil {
		t.Fatalf("reset postgres test database: %v", err)
	}
	contract.Run(t, postgres.NewRepository(pool))
}
