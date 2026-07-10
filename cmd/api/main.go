// Package main provides the CLI entrypoint for running the localhost admin API server.
package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/azusachino/felicia"
	"github.com/azusachino/felicia/internal/api"
	"github.com/azusachino/felicia/internal/domain"
	"github.com/azusachino/felicia/internal/store/pg"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		return errors.New("DATABASE_DSN environment variable is required")
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
	cacheManager := api.NewCacheManager(cacheAddr, logger)

	ctx := context.Background()

	// 2. Initialize PostgreSQL 18 connection pool
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	// 3. Create repository and server
	repo := pg.NewRepository(pool)
	server := api.NewServer(repo, registry, cacheManager, logger)

	// 4. Start local admin web server
	logger.Info("starting admin server", "url", "http://localhost:"+port)
	return http.ListenAndServe(":"+port, server.Handler())
}
