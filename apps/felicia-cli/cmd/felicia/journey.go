package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/local"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	"github.com/azusachino/felicia/apps/felicia-runtime/intake"
	"github.com/azusachino/felicia/apps/felicia-runtime/workspace"
)

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
