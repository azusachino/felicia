// Package main implements the user-facing Felicia CLI.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/core/domain"
	journeypackage "github.com/azusachino/felicia/core/package"
	"github.com/azusachino/felicia/providers/local"
	"github.com/azusachino/felicia/providers/sqlite"
	"github.com/azusachino/felicia/publication"
	"github.com/azusachino/felicia/runtime/importer"
	"github.com/azusachino/felicia/runtime/intake"
)

func main() {
	if err := execute(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "felicia:", err)
		os.Exit(1)
	}
}

func execute(args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: felicia-cli package|import|journey|static")
	}
	switch args[0] {
	case "package":
		return packageCommand(args[1:], output)
	case "import":
		return importCommand(args[1:], output)
	case "journey":
		return journeyCommand(args[1:], output)
	case "static":
		if len(args) < 2 || args[1] != "compile" {
			return errors.New("usage: felicia-cli static compile [options]")
		}
		return compileCommand(args[2:], output)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func journeyCommand(args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: felicia-cli journey plan|apply|review")
	}
	switch args[0] {
	case "plan":
		return journeyPlanCommand(args[1:], output)
	case "apply":
		return journeyApplyCommand(args[1:], output)
	case "review":
		return journeyReviewCommand(args[1:], output)
	default:
		return fmt.Errorf("unknown journey command %q", args[0])
	}
}

func journeyPlanCommand(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("journey plan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	journeyID := flags.String("journey", "", "journey UUID")
	gpxPath := flags.String("gpx", "", "local GPX path")
	photosPath := flags.String("photos", "", "local media directory")
	from := flags.String("from", "", "RFC3339 range start")
	to := flags.String("to", "", "RFC3339 range end")
	format := flags.String("format", "json", "json or jsonl")
	if err := flags.Parse(args); err != nil {
		return err
	}
	id, err := uuid.Parse(*journeyID)
	if err != nil {
		return errors.New("--journey must be a valid UUID")
	}
	if *gpxPath == "" {
		return errors.New("--gpx is required")
	}
	start, err := parseOptionalTime(*from)
	if err != nil {
		return fmt.Errorf("--from: %w", err)
	}
	end, err := parseOptionalTime(*to)
	if err != nil {
		return fmt.Errorf("--to: %w", err)
	}
	var media domain.PhotoSource
	if *photosPath != "" {
		media = local.NewPhotoSource(*photosPath)
	}
	fingerprint, err := fileFingerprint(*gpxPath)
	if err != nil {
		return err
	}
	plan, err := intake.NewService(nil).Plan(context.Background(), intake.PlanRequest{JourneyID: id, From: start, To: end, SourceFingerprint: fingerprint, Sources: intake.SourceSet{Routes: local.NewGPXSource(*gpxPath), Media: media}})
	if err != nil {
		return err
	}
	if *format == "jsonl" {
		return writePlanJSONL(output, plan)
	}
	if *format != "json" {
		return errors.New("--format must be json or jsonl")
	}
	return writeJSON(output, plan)
}

func journeyApplyCommand(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("journey apply", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	database := flags.String("db", "", "SQLite database path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *database == "" || flags.NArg() != 1 {
		return errors.New("usage: felicia-cli journey apply --db <path> <plan.json>")
	}
	data, err := os.ReadFile(flags.Arg(0))
	if err != nil {
		return err
	}
	var plan intake.DraftPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return fmt.Errorf("decode plan: %w", err)
	}
	repo, err := sqlite.Open(*database)
	if err != nil {
		return err
	}
	defer func() { _ = repo.Close() }()
	if err := intake.NewService(repo).Apply(context.Background(), plan); err != nil {
		return err
	}
	return writeJSON(output, map[string]any{"schema": intake.PlanSchema, "mode": "apply", "journey_id": plan.JourneyID, "stops": len(plan.Stops)})
}

func journeyReviewCommand(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("journey review", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	database := flags.String("db", "", "SQLite database path")
	candidateID := flags.String("candidate", "", "candidate UUID")
	state := flags.String("state", "", "proposed, kept, ignored, or merged")
	label := flags.String("label", "", "review label")
	expected := flags.Int64("expected-revision", 0, "expected candidate revision")
	if err := flags.Parse(args); err != nil {
		return err
	}
	id, err := uuid.Parse(*candidateID)
	if err != nil {
		return errors.New("--candidate must be a valid UUID")
	}
	if *database == "" || *state == "" {
		return errors.New("--db and --state are required")
	}
	repo, err := sqlite.Open(*database)
	if err != nil {
		return err
	}
	defer func() { _ = repo.Close() }()
	patch := &domain.StopReviewPatch{CandidateID: id, State: domain.CandidateState(*state)}
	if flags.Lookup("label").Value.String() != "" {
		patch.Label = label
	}
	if *expected > 0 {
		patch.ExpectedRevision = expected
	}
	if err := intake.NewService(repo).Review(context.Background(), patch); err != nil {
		return err
	}
	candidate, err := repo.GetStopCandidate(context.Background(), id)
	if err != nil {
		return err
	}
	return writeJSON(output, candidate)
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func fileFingerprint(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("open source: %w", err)
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
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

func packageCommand(args []string, output io.Writer) error {
	if len(args) != 2 || args[0] != "validate" {
		return errors.New("usage: felicia-cli package validate <journey.zip>")
	}
	pkg, err := journeypackage.Read(args[1])
	if err != nil {
		return err
	}
	return writeJSON(output, map[string]any{"package_id": pkg.Manifest.PackageID, "schema_version": pkg.Manifest.SchemaVersion, "files": len(pkg.Files)})
}

func importCommand(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("import", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	database := flags.String("db", "", "SQLite database path")
	mediaRoot := flags.String("media-root", ".felicia/media", "private local media root")
	apply := flags.Bool("apply", false, "write the package to SQLite and copy media")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: felicia-cli import [--db path] [--media-root path] [--apply] <journey.zip>")
	}
	pkg, err := journeypackage.Read(flags.Arg(0))
	if err != nil {
		return err
	}
	document, err := importer.DecodePackage(pkg)
	if err != nil {
		return err
	}
	if !*apply {
		return writeJSON(output, map[string]any{"mode": "dry-run", "package_id": pkg.Manifest.PackageID, "journeys": 1, "mementos": len(document.Mementos), "photos": len(document.Photos)})
	}
	if *database == "" {
		return errors.New("--db is required with --apply")
	}
	repo, err := sqlite.Open(*database)
	if err != nil {
		return err
	}
	defer func() { _ = repo.Close() }()
	report, err := importer.ApplyPackage(context.Background(), document, repo)
	if err != nil {
		return err
	}
	for filename, data := range pkg.Files {
		if !strings.HasPrefix(filename, "media/") {
			continue
		}
		destination, err := safePath(*mediaRoot, filename)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(destination, data, 0o600); err != nil {
			return err
		}
	}
	return writeJSON(output, map[string]any{"mode": "apply", "package_id": pkg.Manifest.PackageID, "journeys": report.Journeys, "mementos": report.Mementos, "photos": report.Photos})
}

func compileCommand(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("static compile", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	database := flags.String("db", "", "SQLite database path")
	mediaRoot := flags.String("media-root", ".felicia/media", "private local media root")
	out := flags.String("out", "site", "static output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *database == "" {
		return errors.New("--db is required")
	}
	repo, err := sqlite.Open(*database)
	if err != nil {
		return err
	}
	defer func() { _ = repo.Close() }()
	writer := &fileArtifactWriter{root: *out}
	report, err := (publication.StaticCompiler{}).Compile(context.Background(), publication.Input{}, repo, fileMediaSource{root: *mediaRoot}, writer)
	if err != nil {
		return err
	}
	return writeJSON(output, report)
}

type fileMediaSource struct{ root string }

func (source fileMediaSource) Open(_ context.Context, objectKey string) (io.ReadCloser, error) {
	filename, err := safePath(source.root, objectKey)
	if err != nil {
		return nil, err
	}
	return os.Open(filename)
}

type fileArtifactWriter struct{ root string }

func (writer *fileArtifactWriter) WriteJSON(filename string, value any) error {
	destination, err := safePath(writer.root, filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(destination, append(data, '\n'), 0o644)
}

func (writer *fileArtifactWriter) WriteMedia(filename string, source io.Reader) error {
	destination, err := safePath(writer.root, filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = io.Copy(file, source)
	return err
}

func safePath(root, relative string) (string, error) {
	if relative == "" || strings.Contains(relative, "\\") || strings.HasPrefix(relative, "/") {
		return "", fmt.Errorf("unsafe relative path %q", relative)
	}
	clean := path.Clean(relative)
	if clean == "." || clean != relative || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe relative path %q", relative)
	}
	return filepath.Join(root, filepath.FromSlash(clean)), nil
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
