// Package main provides the CLI entrypoint for running the localhost admin API server.
package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/azusachino/felicia"
	"github.com/azusachino/felicia/internal/api"
	"github.com/azusachino/felicia/internal/domain"
	"github.com/azusachino/felicia/internal/store/pg"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("DATABASE_DSN environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 1. Load kind templates registry from embedded filesystem first
	subFS, err := fs.Sub(felicia.KindsFS, "kinds")
	if err != nil {
		return err
	}
	registry, err := domain.LoadRegistry(subFS)
	if err != nil {
		return err
	}

	cacheAddr := os.Getenv("CACHE_ADDR")
	if cacheAddr == "" {
		cacheAddr = "localhost:6379"
	}
	cacheManager := api.NewCacheManager(cacheAddr)

	ctx := context.Background()

	// 2. Initialize PostgreSQL 18 connection pool
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	// 3. Create repository and server
	repo := pg.NewRepository(pool)
	server := api.NewServer(repo, registry, cacheManager)

	// 4. Start local admin web server
	log.Printf("Starting felicia local admin server on http://localhost:%s", port)
	return http.ListenAndServe(":"+port, server.Handler())
}
