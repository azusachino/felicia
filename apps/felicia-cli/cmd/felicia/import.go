package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	journeypackage "github.com/azusachino/felicia/apps/felicia-core/journeypackage"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
	"github.com/azusachino/felicia/apps/felicia-runtime/importer"
)

func newImportCmd() *cobra.Command {
	var (
		database  string
		mediaRoot string
		apply     bool
	)

	importCmd := &cobra.Command{
		Use:   "import [--db <path>] [--media-root <path>] [--apply] <journey.zip>",
		Short: "Import a journey package archive into SQLite and local media store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg, err := journeypackage.Read(args[0])
			if err != nil {
				return err
			}
			document, err := importer.DecodePackage(pkg)
			if err != nil {
				return err
			}
			registry, err := importer.DefaultRegistry()
			if err != nil {
				return err
			}
			if err := importer.ValidatePackageDocument(document, registry); err != nil {
				return err
			}
			if !apply {
				return writeJSON(cmd.OutOrStdout(), map[string]any{
					"mode":       "dry-run",
					"package_id": pkg.Manifest.PackageID,
					"journeys":   1,
					"candidates": len(document.Stops),
					"mementos":   len(document.Mementos),
					"photos":     len(document.Photos),
				})
			}
			if err := resolveWorkspaceDefaults(&database, &mediaRoot); err != nil {
				return err
			}
			repo, err := sqlite.Open(database)
			if err != nil {
				return err
			}
			defer func() { _ = repo.Close() }()

			if err := installPackageMedia(pkg, mediaRoot); err != nil {
				return err
			}
			report, err := importer.ApplyPackage(context.Background(), document, repo)
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"mode":                 "apply",
				"package_id":           pkg.Manifest.PackageID,
				"journeys":             report.Journeys,
				"candidates":           report.Candidates,
				"mementos":             report.Mementos,
				"photos":               report.Photos,
				"authorship_conflicts": report.Conflicts,
			})
		},
	}

	importCmd.Flags().StringVar(&database, "db", "", "SQLite database path")
	importCmd.Flags().StringVar(&mediaRoot, "media-root", "", "private local media root")
	importCmd.Flags().BoolVar(&apply, "apply", false, "write the package to SQLite and copy media")

	return importCmd
}

// installPackageMedia writes every media member into the media root under its
// content identity. Content addressing makes this idempotent: the path encodes
// the digest, so a member already present is already the right bytes and is
// left alone. Publishing only ever happens by rename, so a file visible at its
// final path is a complete one -- an interrupted run leaves a discarded
// temporary file rather than a truncated original.
func installPackageMedia(pkg *journeypackage.Package, mediaRoot string) error {
	for filename, data := range pkg.Files {
		if !strings.HasPrefix(filename, "media/") {
			continue
		}
		// The same derivation the importer recorded, so a member name shared
		// with another package cannot overwrite that package's bytes (ADR-0026).
		destination, err := publication.SafeJoin(mediaRoot, importer.MediaObjectKey(filename, importer.MediaDigest(data)))
		if err != nil {
			return err
		}
		if _, err := os.Stat(destination); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		if err := writeOriginalAtomically(destination, data); err != nil {
			return fmt.Errorf("install %s: %w", filename, err)
		}
	}
	return nil
}

func writeOriginalAtomically(destination string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(destination), ".felicia-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temp.Name()) }()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(temp.Name(), destination)
}
