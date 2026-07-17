// Package main implements the user-facing Felicia CLI.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	journeypackage "github.com/azusachino/felicia/core/package"
	"github.com/azusachino/felicia/providers/sqlite"
	"github.com/azusachino/felicia/publication"
	"github.com/azusachino/felicia/runtime/importer"
)

func main() {
	if err := execute(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "felicia:", err)
		os.Exit(1)
	}
}

func execute(args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: felicia-cli package validate|import|static compile")
	}
	switch args[0] {
	case "package":
		return packageCommand(args[1:], output)
	case "import":
		return importCommand(args[1:], output)
	case "static":
		if len(args) < 2 || args[1] != "compile" {
			return errors.New("usage: felicia-cli static compile [options]")
		}
		return compileCommand(args[2:], output)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
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
