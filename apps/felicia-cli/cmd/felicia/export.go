package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
)

func newExportCmd() *cobra.Command {
	var (
		journeySlug string
		dir         string
		out         string
		database    string
		deferAssets bool
	)

	cmd := &cobra.Command{
		Use:   "export [--journey <slug>] [--dir <path>] [options]",
		Short: "Export a published journey as a sanitized GeoJSON FeatureCollection",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if journeySlug == "" && dir == "" {
				return errors.New("usage: felicia-cli export --journey <slug> | --dir <path> [--out <path>]")
			}

			opts := publication.ExportOptions{
				DeferAssets: deferAssets,
			}

			var (
				col *publication.GeoJSONFeatureCollection
				err error
			)

			if dir != "" {
				col, err = publication.ExportJourneyFolderGeoJSON(dir, opts)
				if err != nil {
					return fmt.Errorf("export journey folder %s: %w", dir, err)
				}
			} else if folder := findJourneyFolder(journeySlug); folder != "" {
				col, err = publication.ExportJourneyFolderGeoJSON(folder, opts)
				if err != nil {
					return fmt.Errorf("export journey folder %s: %w", folder, err)
				}
			} else {
				if database == "" {
					if err := resolveWorkspaceDefaults(&database, nil); err != nil {
						return err
					}
				}
				repo, err := sqlite.Open(database)
				if err != nil {
					return fmt.Errorf("open sqlite %s: %w", database, err)
				}
				defer func() { _ = repo.Close() }()

				col, err = publication.ExportJourneyGeoJSON(cmd.Context(), journeySlug, repo, opts)
				if err != nil {
					return fmt.Errorf("export journey %s: %w", journeySlug, err)
				}
			}

			data, err := publication.MarshalGeoJSON(col)
			if err != nil {
				return fmt.Errorf("marshal geojson: %w", err)
			}

			if out == "-" {
				_, err := cmd.OutOrStdout().Write(data)
				return err
			}

			targetPath := out
			if targetPath == "" || targetPath == "." {
				targetPath = fmt.Sprintf("%s.geojson", col.Properties.Slug)
			} else {
				if stat, statErr := os.Stat(targetPath); (statErr == nil && stat.IsDir()) || strings.HasSuffix(targetPath, "/") {
					targetPath = filepath.Join(targetPath, fmt.Sprintf("%s.geojson", col.Properties.Slug))
				}
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("create output directory: %w", err)
			}

			if err := os.WriteFile(targetPath, data, 0o644); err != nil {
				return fmt.Errorf("write geojson %s: %w", targetPath, err)
			}

			// Count features for summary
			routePoints := 0
			stopCount := 0
			mementoCount := 0
			for _, f := range col.Features {
				switch f.Properties["role"] {
				case "route":
					if geom, ok := f.Geometry.(map[string]any); ok {
						if coords, ok := geom["coordinates"].([][]float64); ok {
							routePoints = len(coords)
						}
					}
				case "stop":
					stopCount++
				case "memento":
					mementoCount++
				}
			}

			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"slug":         col.Properties.Slug,
				"title":        col.Properties.Title,
				"route_points": routePoints,
				"stops":        stopCount,
				"mementos":     mementoCount,
				"out":          targetPath,
			})
		},
	}

	cmd.Flags().StringVar(&journeySlug, "journey", "", "journey slug or ID")
	cmd.Flags().StringVar(&dir, "dir", "", "path to journey folder containing journey.json, stops.json, etc.")
	cmd.Flags().StringVarP(&out, "out", "o", ".", "output file or directory path (use '-' for stdout)")
	cmd.Flags().StringVar(&database, "db", "", "SQLite database path")
	cmd.Flags().BoolVar(&deferAssets, "defer-assets", false, "defer external media resolution and emit empty media arrays")

	return cmd
}

func findJourneyFolder(journeySlug string) string {
	for _, base := range []string{
		".",
		filepath.Join("vendor", "felicia"),
		filepath.Join("..", "..", ".."),
		filepath.Join("..", ".."),
		"..",
		filepath.Join("..", "..", "..", ".."),
	} {
		candidate := filepath.Join(base, "publication", "journeys", journeySlug)
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate
		}
		direct := filepath.Join(base, journeySlug)
		if stat, err := os.Stat(direct); err == nil && stat.IsDir() {
			return direct
		}
	}
	return ""
}
