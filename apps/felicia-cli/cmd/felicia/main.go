// Package main implements the user-facing Felicia CLI.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	journeypackage "github.com/azusachino/felicia/apps/felicia-core/journeypackage"
	"github.com/azusachino/felicia/apps/felicia-providers/local"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
	"github.com/azusachino/felicia/apps/felicia-runtime/importer"
	"github.com/azusachino/felicia/apps/felicia-runtime/intake"
	"github.com/azusachino/felicia/apps/felicia-runtime/workspace"
)

func main() {
	if err := execute(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "felicia:", err)
		os.Exit(1)
	}
}

func execute(args []string, output io.Writer) error {
	cmd := newRootCmd(output)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func newRootCmd(output io.Writer) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "felicia-cli",
		Short:         "Felicia CLI for travel journal intake, publication, and package management",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("usage: felicia-cli package|import|journey|static")
		},
	}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)

	rootCmd.AddCommand(newPackageCmd())
	rootCmd.AddCommand(newImportCmd())
	rootCmd.AddCommand(newJourneyCmd())
	rootCmd.AddCommand(newStaticCmd())

	return rootCmd
}

func newPackageCmd() *cobra.Command {
	packageCmd := &cobra.Command{
		Use:   "package",
		Short: "Validate and inspect journey packages",
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("usage: felicia-cli package validate <journey.zip>")
		},
	}

	validateCmd := &cobra.Command{
		Use:   "validate <journey.zip>",
		Short: "Validate a journey package archive and its manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg, err := journeypackage.Read(args[0])
			if err != nil {
				return err
			}
			document, err := importer.DecodePackage(pkg)
			if err != nil {
				return fmt.Errorf("validate package contents: %w", err)
			}
			registry, err := importer.DefaultRegistry()
			if err != nil {
				return fmt.Errorf("load kind registry: %w", err)
			}
			if err := importer.ValidatePackageDocument(document, registry); err != nil {
				return fmt.Errorf("validate package contents: %w", err)
			}
			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"package_id":     pkg.Manifest.PackageID,
				"schema_version": pkg.Manifest.SchemaVersion,
				"files":          len(pkg.Files),
				"journeys":       1,
				"candidates":     len(document.Stops),
				"mementos":       len(document.Mementos),
				"photos":         len(document.Photos),
			})
		},
	}

	packageCmd.AddCommand(validateCmd)
	return packageCmd
}

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

func newJourneyCmd() *cobra.Command {
	journeyCmd := &cobra.Command{
		Use:   "journey",
		Short: "Manage journey intake, planning, review, and authoring",
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("usage: felicia-cli journey ingest|plan|apply|review|delete")
		},
	}

	journeyCmd.AddCommand(newJourneyIngestCmd())
	journeyCmd.AddCommand(newJourneyPlanCmd())
	journeyCmd.AddCommand(newJourneyApplyCmd())
	journeyCmd.AddCommand(newJourneyReviewCmd())
	journeyCmd.AddCommand(newJourneyDeleteCmd())

	return journeyCmd
}

func newJourneyIngestCmd() *cobra.Command {
	var (
		dir       string
		wsPath    string
		database  string
		mediaRoot string
		slug      string
		title     string
		place     string
		from      string
		to        string
	)

	cmd := &cobra.Command{
		Use:   "ingest [--dir <path>] [options] [<path>]",
		Short: "Single-step trip folder ingestion and draft staging",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir := dir
			if targetDir == "" && len(args) > 0 {
				targetDir = args[0]
			}
			if targetDir == "" {
				return errors.New("usage: felicia-cli journey ingest [--dir] <path> [options]")
			}

			ws, err := workspace.Resolve(wsPath)
			if err != nil {
				return fmt.Errorf("resolve workspace: %w", err)
			}
			if err := ws.EnsureDirs(); err != nil {
				return err
			}
			if database == "" {
				database = ws.Database
			}
			if mediaRoot == "" {
				mediaRoot = ws.MediaRoot
			}
			if err := os.MkdirAll(filepath.Dir(database), 0o755); err != nil {
				return err
			}
			if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
				return err
			}

			repo, err := sqlite.Open(database)
			if err != nil {
				return err
			}
			defer func() { _ = repo.Close() }()

			startTime, err := parseOptionalTime(from)
			if err != nil {
				return fmt.Errorf("--from: %w", err)
			}
			endTime, err := parseOptionalTime(to)
			if err != nil {
				return fmt.Errorf("--to: %w", err)
			}

			cfg := TripFolderConfig{
				Dir:           targetDir,
				Slug:          slug,
				Title:         title,
				Place:         place,
				WorkspaceRoot: ws.Root,
				Database:      database,
				MediaRoot:     mediaRoot,
				From:          startTime,
				To:            endTime,
			}

			report, err := IngestTripFolder(context.Background(), cfg, repo, mediaRoot)
			if err != nil {
				return err
			}

			conflicts := report.Conflicts
			if conflicts == nil {
				conflicts = []string{}
			}

			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"mode":       "ingest",
				"journey_id": report.JourneyID.String(),
				"slug":       report.Slug,
				"candidates": report.Candidates,
				"mementos":   report.Mementos,
				"photos":     report.Photos,
				"conflicts":  conflicts,
			})
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "path to trip folder")
	cmd.Flags().StringVar(&wsPath, "workspace", "", "workspace root directory")
	cmd.Flags().StringVar(&database, "db", "", "SQLite database path")
	cmd.Flags().StringVar(&mediaRoot, "media-root", "", "private local media root")
	cmd.Flags().StringVar(&slug, "slug", "", "journey slug")
	cmd.Flags().StringVar(&title, "title", "", "journey title")
	cmd.Flags().StringVar(&place, "place", "", "journey place")
	cmd.Flags().StringVar(&from, "from", "", "RFC3339 range start")
	cmd.Flags().StringVar(&to, "to", "", "RFC3339 range end")

	return cmd
}

func newJourneyPlanCmd() *cobra.Command {
	var (
		journeyID string
		gpxPath   string
		timeline  string
		photos    string
		sidecar   string
		from      string
		to        string
		format    string
	)

	cmd := &cobra.Command{
		Use:   "plan --journey <uuid> [--gpx <path>] [--timeline <path>] [options]",
		Short: "Plan journey stops and draft mementos from route or timeline sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, err := uuid.Parse(journeyID)
			if err != nil {
				return errors.New("--journey must be a valid UUID")
			}
			if gpxPath == "" && timeline == "" {
				return errors.New("--gpx or --timeline is required")
			}
			start, err := parseOptionalTime(from)
			if err != nil {
				return fmt.Errorf("--from: %w", err)
			}
			end, err := parseOptionalTime(to)
			if err != nil {
				return fmt.Errorf("--to: %w", err)
			}
			var media domain.PhotoSource
			if photos != "" {
				media = local.NewPhotoSourceWithSidecar(photos, sidecar)
			}
			var routes domain.RouteSource
			if gpxPath != "" {
				routes = local.NewGPXSource(gpxPath)
			}
			var visits domain.VisitSource
			if timeline != "" {
				visits = local.NewTimelineSource(timeline)
			}
			fingerprint, err := sourceFingerprint(gpxPath, timeline)
			if err != nil {
				return err
			}
			plan, err := intake.NewService(nil, nil).Plan(context.Background(), intake.PlanRequest{
				JourneyID:         id,
				From:              start,
				To:                end,
				SourceFingerprint: fingerprint,
				Sources:           intake.SourceSet{Routes: routes, Visits: visits, Media: media},
			})
			if err != nil {
				return err
			}
			if format == "jsonl" {
				return writePlanJSONL(cmd.OutOrStdout(), plan)
			}
			if format != "json" {
				return errors.New("--format must be json or jsonl")
			}
			return writeJSON(cmd.OutOrStdout(), plan)
		},
	}

	cmd.Flags().StringVar(&journeyID, "journey", "", "journey UUID")
	cmd.Flags().StringVar(&gpxPath, "gpx", "", "local GPX path")
	cmd.Flags().StringVar(&timeline, "timeline", "", "Google Timeline JSON export")
	cmd.Flags().StringVar(&photos, "photos", "", "local media directory")
	cmd.Flags().StringVar(&sidecar, "sidecar", "", "local photo JSONL sidecar")
	cmd.Flags().StringVar(&from, "from", "", "RFC3339 range start")
	cmd.Flags().StringVar(&to, "to", "", "RFC3339 range end")
	cmd.Flags().StringVar(&format, "format", "json", "json or jsonl")

	return cmd
}

func newJourneyApplyCmd() *cobra.Command {
	var database string

	cmd := &cobra.Command{
		Use:   "apply [--db <path>] <plan.json>",
		Short: "Apply a drafted journey plan to SQLite",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := resolveWorkspaceDefaults(&database, nil); err != nil {
				return err
			}
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			var plan intake.DraftPlan
			if err := json.Unmarshal(data, &plan); err != nil {
				return fmt.Errorf("decode plan: %w", err)
			}
			repo, err := sqlite.Open(database)
			if err != nil {
				return err
			}
			defer func() { _ = repo.Close() }()
			if err := intake.NewService(repo, repo).Apply(context.Background(), plan); err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"schema":     intake.PlanSchema,
				"mode":       "apply",
				"journey_id": plan.JourneyID,
				"stops":      len(plan.Stops),
			})
		},
	}

	cmd.Flags().StringVar(&database, "db", "", "SQLite database path")
	return cmd
}

func newJourneyReviewCmd() *cobra.Command {
	var (
		database    string
		candidateID string
		state       string
		label       string
		expected    int64
	)

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Review and curate a stop candidate",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, err := uuid.Parse(candidateID)
			if err != nil {
				return errors.New("--candidate must be a valid UUID")
			}
			if state == "" {
				return errors.New("--state is required")
			}
			if err := resolveWorkspaceDefaults(&database, nil); err != nil {
				return err
			}
			repo, err := sqlite.Open(database)
			if err != nil {
				return err
			}
			defer func() { _ = repo.Close() }()
			patch := &domain.StopReviewPatch{CandidateID: id, State: domain.CandidateState(state)}
			if cmd.Flags().Changed("label") {
				patch.Label = &label
			}
			if expected > 0 {
				patch.ExpectedRevision = &expected
			}
			if err := intake.NewService(repo, repo).Review(context.Background(), patch); err != nil {
				return err
			}
			candidate, err := repo.GetStopCandidate(context.Background(), id)
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), candidate)
		},
	}

	cmd.Flags().StringVar(&database, "db", "", "SQLite database path")
	cmd.Flags().StringVar(&candidateID, "candidate", "", "candidate UUID")
	cmd.Flags().StringVar(&state, "state", "", "proposed, kept, ignored, or merged")
	cmd.Flags().StringVar(&label, "label", "", "review label")
	cmd.Flags().Int64Var(&expected, "expected-revision", 0, "expected candidate revision")

	return cmd
}

func newJourneyDeleteCmd() *cobra.Command {
	var (
		database  string
		journeyID string
	)

	cmd := &cobra.Command{
		Use:   "delete [--db <path>] --journey <uuid>",
		Short: "Delete a journey and its cascaded entities from SQLite",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if journeyID == "" {
				return errors.New("usage: felicia-cli journey delete [--db <path>] --journey <uuid>")
			}
			if err := resolveWorkspaceDefaults(&database, nil); err != nil {
				return err
			}
			id, err := uuid.Parse(journeyID)
			if err != nil {
				return fmt.Errorf("invalid journey UUID %q: %w", journeyID, err)
			}
			repo, err := sqlite.Open(database)
			if err != nil {
				return err
			}
			defer func() { _ = repo.Close() }()
			if err := repo.DeleteJourney(context.Background(), id); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "deleted journey %s\n", id)
			return err
		},
	}

	cmd.Flags().StringVar(&database, "db", "", "SQLite database path")
	cmd.Flags().StringVar(&journeyID, "journey", "", "journey UUID")

	return cmd
}

func newStaticCmd() *cobra.Command {
	staticCmd := &cobra.Command{
		Use:   "static",
		Short: "Static site compilation",
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("usage: felicia-cli static compile [options]")
		},
	}

	var (
		database  string
		mediaRoot string
		out       string
	)

	compileCmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile static site publication from SQLite and media root",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := resolveWorkspaceDefaults(&database, &mediaRoot); err != nil {
				return err
			}
			repo, err := sqlite.Open(database)
			if err != nil {
				return err
			}
			defer func() { _ = repo.Close() }()
			writer := &publication.FileArtifactWriter{Root: out}
			report, err := (publication.StaticCompiler{}).Compile(
				context.Background(),
				publication.Input{},
				repo,
				publication.FileMediaSource{Root: mediaRoot},
				writer,
			)
			if err != nil {
				_ = writer.Abort()
				return err
			}
			removed, err := writer.Finalize()
			if err != nil {
				_ = writer.Abort()
				return err
			}
			report.Removed = len(removed)
			return writeJSON(cmd.OutOrStdout(), report)
		},
	}

	compileCmd.Flags().StringVar(&database, "db", "", "SQLite database path")
	compileCmd.Flags().StringVar(&mediaRoot, "media-root", "", "private local media root")
	compileCmd.Flags().StringVar(&out, "out", "site", "static output directory")

	staticCmd.AddCommand(compileCmd)
	return staticCmd
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func sourceFingerprint(filenames ...string) (string, error) {
	hash := sha256.New()
	for _, filename := range filenames {
		if filename == "" {
			continue
		}
		file, err := os.Open(filename)
		if err != nil {
			return "", fmt.Errorf("open source %s: %w", filename, err)
		}
		if _, err := io.WriteString(hash, filename+"\x00"); err != nil {
			_ = file.Close()
			return "", err
		}
		if _, err := io.Copy(hash, file); err != nil {
			_ = file.Close()
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
	}
	return "sha256:" + fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func writePlanJSONL(output io.Writer, plan intake.DraftPlan) error {
	encoder := json.NewEncoder(output)
	for _, stop := range plan.Stops {
		if err := encoder.Encode(map[string]any{"type": "stop", "stop": stop}); err != nil {
			return err
		}
	}
	return encoder.Encode(map[string]any{"type": "summary", "schema": plan.Schema, "version": plan.Version, "source_fingerprint": plan.SourceFingerprint, "stops": len(plan.Stops), "mementos": len(plan.Mementos), "issues": len(plan.Issues)})
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

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func resolveWorkspaceDefaults(database, mediaRoot *string) error {
	if (database != nil && *database == "") || (mediaRoot != nil && *mediaRoot == "") {
		ws, err := workspace.Resolve("")
		if err != nil {
			return fmt.Errorf("resolve workspace: %w", err)
		}
		if err := ws.EnsureDirs(); err != nil {
			return err
		}
		if database != nil && *database == "" {
			*database = ws.Database
		}
		if mediaRoot != nil && *mediaRoot == "" {
			*mediaRoot = ws.MediaRoot
		}
	}
	return nil
}
