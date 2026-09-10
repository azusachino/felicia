// Package main implements a PostgreSQL-backed static site compiler.
//
// NOT PART OF V1. This entry point composes the PostgreSQL provider, which
// ADR-0032 defers to v1.1/v1.2, so it is experimental deferred-provider work
// and no v1 workflow, gate or deployment invokes it. It does use the shared
// publication.StaticCompiler, so it is not a second compiler -- only a second
// composition root, and provider choice per composition root is what ADR-0028
// permits.
//
// The supported path is `felicia-cli static compile`, which does the same job
// against SQLite. Prefer that; reach for this only while working on
// PostgreSQL re-entry.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/azusachino/felicia/apps/felicia-providers/postgres"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
)

func main() {
	// The build tool writes files to the output dir, so logs go to stderr.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := run(); err != nil {
		slog.Error("static site compilation failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := flag.String("dsn", os.Getenv("DATABASE_DSN"), "PostgreSQL 18 connection DSN")
	outDir := flag.String("out", "dist", "Output directory for static files")
	mediaRoot := flag.String("media-root", ".felicia/media", "Source media root for photos referenced by published mementos")
	flag.Parse()

	if *dsn == "" {
		return fmt.Errorf("database DSN is required. Set DATABASE_DSN env var or pass -dsn flag")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	repo := postgres.NewRepository(pool)

	slog.Info("starting static site compilation", "out_dir", *outDir)

	writer := &publication.FileArtifactWriter{Root: *outDir}
	report, err := (publication.StaticCompiler{}).Compile(ctx, publication.Input{}, repo, publication.FileMediaSource{Root: *mediaRoot}, writer)
	if err != nil {
		return err
	}
	// Reconcile a reused output directory: unpublished or deleted content
	// from a previous compile must not stay publicly reachable.
	removed, err := writer.Finalize()
	if err != nil {
		return err
	}
	report.Removed = len(removed)

	slog.Info("static site compilation complete", "out_dir", *outDir,
		"journeys", report.Journeys, "mementos", report.Mementos,
		"media", report.Media, "removed", report.Removed)
	return nil
}
