// Package main provides the entrypoint for Felicia Desktop Studio.
//
// Security posture: the privileged admin surface is served only through the
// app's own webview custom scheme (no TCP listener); the optional session
// token in HandlerConfig exists for tests. The local preview server is a
// read-only loopback listener for published artifacts only.
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
	"github.com/azusachino/felicia/apps/felicia-runtime/workspace"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "felicia-desktop: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	mode := flag.String("mode", "admin", "desktop mode: admin or reader")
	sample := flag.Bool("sample", false, "open a fresh isolated synthetic sample trip")
	dbPath := flag.String("db", "", "path to SQLite database (default ~/.felicia/felicia.sqlite)")
	mediaPath := flag.String("media-root", "", "path to media root (default ~/.felicia/media)")
	publicPath := flag.String("public-dir", "", "path to public static site output (default ~/.felicia/site)")
	previewAddr := flag.String("preview-addr", "127.0.0.1:8081", "loopback address for the read-only site preview server (admin mode)")
	flag.Parse()

	if *mode != "admin" && *mode != "reader" {
		return fmt.Errorf("invalid mode %q: must be 'admin' or 'reader'", *mode)
	}

	if *sample {
		if *mode != "admin" || *dbPath != "" || *mediaPath != "" || *publicPath != "" {
			return fmt.Errorf("sample mode cannot be combined with reader mode or workspace paths")
		}
		root, err := os.MkdirTemp("", "felicia-empty-")
		if err != nil {
			return fmt.Errorf("create isolated empty workspace: %w", err)
		}
		defer func() { _ = os.RemoveAll(root) }()
		*dbPath, *mediaPath, *publicPath = filepath.Join(root, "felicia.sqlite"), filepath.Join(root, "media"), filepath.Join(root, "site")
		*previewAddr = "127.0.0.1:0"
	}

	if err := validateE2EPaths(*dbPath, *mediaPath, *publicPath); err != nil {
		return err
	}

	workspace := ""
	if !*sample {
		var err error
		workspace, err = resolveDefaultWorkspace()
		if err != nil {
			return fmt.Errorf("resolve workspace: %w", err)
		}
	}

	actualDB := *dbPath
	if actualDB == "" {
		actualDB = filepath.Join(workspace, "felicia.sqlite")
	}
	actualMedia := *mediaPath
	if actualMedia == "" {
		actualMedia = filepath.Join(workspace, "media")
	}
	actualPublic := *publicPath
	if actualPublic == "" {
		actualPublic = filepath.Join(workspace, "site")
	}

	for _, dir := range []string{filepath.Dir(actualDB), actualMedia, actualPublic} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	repo, err := sqlite.Open(actualDB)
	if err != nil {
		return fmt.Errorf("open sqlite repository: %w", err)
	}
	defer func() { _ = repo.Close() }()

	// Ensure the sole journal exists.
	ctx := context.Background()
	if _, err := repo.GetSoleJournal(ctx); err != nil {
		_ = repo.CreateJournal(ctx, &domain.Journal{ID: uuid.Must(uuid.NewV7()), CreatedAt: time.Now().UTC()})
	}

	// In reader mode, compile once when no artifact manifest exists so the
	// reader has data to show.
	if *mode == "reader" {
		manifestFile := filepath.Join(actualPublic, filepath.FromSlash(publication.ManifestPath))
		if _, err := os.Stat(manifestFile); err != nil {
			writer := &publication.FileArtifactWriter{Root: actualPublic}
			media := publication.FileMediaSource{Root: actualMedia}
			_, _ = (publication.StaticCompiler{}).Compile(ctx, publication.Input{}, repo, media, writer)
			_, _ = writer.Finalize()
		}
	}

	var app *application.App
	var window *application.WebviewWindow

	onPick := func(directories, files bool, title string) (string, error) {
		if app == nil || window == nil {
			return "", fmt.Errorf("desktop window is not initialized")
		}
		return app.Dialog.OpenFile().
			CanChooseDirectories(directories).
			CanChooseFiles(files).
			CanCreateDirectories(directories).
			SetTitle(title).
			AttachToWindow(window).
			PromptForSingleSelection()
	}
	onPickFolder := func(title string) (string, error) { return onPick(true, false, title) }
	onPickFile := func(title string) (string, error) { return onPick(false, true, title) }

	var previewServer *PreviewServer
	if *mode == "admin" {
		readerSub, _ := fs.Sub(embeddedAssets, "assets/reader")
		ps, err := StartPreviewServer(*previewAddr, func() string { return actualPublic }, readerSub)
		if err == nil {
			previewServer = ps
			defer func() { _ = previewServer.Close() }()
		}
	}

	handler, err := NewHandler(HandlerConfig{
		Repo:      repo,
		MediaRoot: actualMedia,
		PublicDir: actualPublic,
		Mode:      *mode,
		Isolated:  *sample,
		PreviewPortFn: func() string {
			if previewServer != nil {
				return previewServer.Port()
			}
			return ""
		},
		OnPickFolder: onPickFolder,
		OnPickFile:   onPickFile,
	})
	if err != nil {
		return fmt.Errorf("initialize handler: %w", err)
	}

	workspaces := NewWorkspaceRouter(handler, func(sample bool) {
		if window != nil {
			title := "Felicia Studio"
			if sample {
				title += " — Sample trip"
			}
			window.SetTitle(title)
		}
	})
	defer workspaces.Close()
	if *sample {
		if err := workspaces.openSample(); err != nil {
			return fmt.Errorf("open isolated sample workspace: %w", err)
		}
	}

	if handled, err := serveE2E(workspaces); handled {
		return err
	}

	app = application.New(application.Options{
		Name:        "Felicia Studio",
		Icon:        defaultAppIcon,
		Description: "Map-based travel journal studio",
		Assets: application.AssetOptions{
			Handler: workspaces,
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyRegular,
		},
	})

	title := "Felicia Studio"
	if *sample {
		title = "Felicia Studio — Sample trip"
	}
	if *mode == "reader" {
		title = "Felicia — Public Reader"
	}

	windowOptions := application.WebviewWindowOptions{
		Title:  title,
		Width:  1100,
		Height: 760,
		URL:    "/",
	}
	applyPlatformWindowChrome(&windowOptions)
	window = app.Window.NewWithOptions(windowOptions)

	return app.Run()
}

func resolveDefaultWorkspace() (string, error) {
	ws, err := workspace.Resolve("")
	if err != nil {
		return "", err
	}
	return ws.Root, nil
}
