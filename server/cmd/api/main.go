// Package main provides the CLI entrypoint for running the localhost admin API server.
package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/azusachino/felicia/core"
	"github.com/azusachino/felicia/core/domain"
	"github.com/azusachino/felicia/providers/dawarich"
	"github.com/azusachino/felicia/providers/immich"
	"github.com/azusachino/felicia/providers/postgres"
	"github.com/azusachino/felicia/providers/sqlite"
	"github.com/azusachino/felicia/runtime/importer"
	"github.com/azusachino/felicia/server/api"
	"github.com/azusachino/felicia/server/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return err
	}

	// 1. Load kind templates registry from embedded filesystem first
	subFS, err := fs.Sub(core.KindsFS, "kinds")
	if err != nil {
		return err
	}
	registry, err := domain.LoadRegistry(subFS)
	if err != nil {
		return err
	}

	cacheManager := api.NewCacheManager(cfg.CacheAddr, logger)

	ctx := context.Background()

	// 2. Initialize the configured provider. SQLite is the local default;
	// PostgreSQL remains available for deployments that need PostGIS.
	var repo domain.Repository
	var closeRepository func()
	if cfg.DatabaseDriver == "sqlite" {
		store, err := sqlite.Open(cfg.DatabasePath)
		if err != nil {
			return err
		}
		repo, closeRepository = store, func() { _ = store.Close() }
	} else {
		pool, err := pgxpool.New(ctx, cfg.DatabaseDSN)
		if err != nil {
			return err
		}
		repo, closeRepository = postgres.NewRepository(pool), pool.Close
	}
	defer closeRepository()

	// 3. Create repository and server

	// Ingest sources are optional; the ingest endpoints return 503 when unset.
	// Point source URLs at a live instance or the mock (scripts/mock_upstream.py).
	var tracks domain.TrackSource
	if cfg.Dawarich.URL != "" {
		tracks = dawarich.New(cfg.Dawarich.URL, cfg.Dawarich.APIKey, nil)
		logger.Info("dawarich source configured", "url", cfg.Dawarich.URL)
	}
	var photos domain.PhotoSource
	if cfg.Immich.URL != "" {
		photos = immich.New(cfg.Immich.URL, cfg.Immich.APIKey, nil)
		logger.Info("immich source configured", "url", cfg.Immich.URL)
	}
	imp := importer.NewWithLogger(tracks, photos, repo, cfg.RDPEpsilon, logger)

	server := api.NewServer(repo, registry, cacheManager, logger, imp, api.RouteConfig{
		TransitSegmentLengthM: cfg.TransitSegmentLenM,
		MediaRoot:             cfg.MediaRoot,
		RatePerSecond:         cfg.RatePerSecond,
		RateBurst:             cfg.RateBurst,
		SiteOutDir:            cfg.SiteOutDir,
		SitePreviewPort:       cfg.SitePreviewPort,
		SiteSpaDist:           cfg.SiteSpaDist,
	})

	// 4. Serve the compiled site on a second local port so the author can
	// verify a build exactly as a static host would deliver it. Losing the
	// preview listener degrades the Site page but must not take the admin
	// API down with it.
	go func() {
		logger.Info("starting site preview server", "url", "http://localhost:"+cfg.SitePreviewPort)
		if err := http.ListenAndServe(":"+cfg.SitePreviewPort, api.PreviewHandler(cfg.SiteOutDir, cfg.SiteSpaDist)); err != nil {
			logger.Error("site preview server stopped", "err", err)
		}
	}()

	// 5. Start local admin web server
	logger.Info("starting admin server", "url", "http://localhost:"+cfg.Port)
	return http.ListenAndServe(":"+cfg.Port, server.Handler())
}
