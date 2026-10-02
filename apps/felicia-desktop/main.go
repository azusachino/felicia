// Package main provides the entrypoint for Felicia Desktop Studio.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "felicia-desktop: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	mode := flag.String("mode", "admin", "desktop mode: admin or reader")
	smoke := flag.Bool("smoke", false, "run automated native smoke verification and exit")
	dbPath := flag.String("db", "", "path to SQLite database (default ~/.felicia/felicia.sqlite)")
	mediaPath := flag.String("media-root", "", "path to media root (default ~/.felicia/media)")
	publicPath := flag.String("public-dir", "", "path to public static site output (default ~/.felicia/site)")
	flag.Parse()

	if *mode != "admin" && *mode != "reader" {
		return fmt.Errorf("invalid mode %q: must be 'admin' or 'reader'", *mode)
	}

	workspace, err := resolveDefaultWorkspace()
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
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

	if err := os.MkdirAll(filepath.Dir(actualDB), 0o755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}
	if err := os.MkdirAll(actualMedia, 0o755); err != nil {
		return fmt.Errorf("create media dir: %w", err)
	}
	if err := os.MkdirAll(actualPublic, 0o755); err != nil {
		return fmt.Errorf("create public dir: %w", err)
	}

	repo, err := sqlite.Open(actualDB)
	if err != nil {
		return fmt.Errorf("open sqlite repository: %w", err)
	}
	defer func() { _ = repo.Close() }()

	// Ensure sole journal exists
	ctx := context.Background()
	if _, err := repo.GetSoleJournal(ctx); err != nil {
		_ = repo.CreateJournal(ctx, &domain.Journal{ID: uuid.Must(uuid.NewV7()), CreatedAt: time.Now().UTC()})
	}

	// In reader mode, compile if manifest is absent so reader has data
	if *mode == "reader" {
		manifestFile := filepath.Join(actualPublic, filepath.FromSlash(publication.ManifestPath))
		if _, err := os.Stat(manifestFile); err != nil {
			writer := &publication.FileArtifactWriter{Root: actualPublic}
			media := publication.FileMediaSource{Root: actualMedia}
			_, _ = (publication.StaticCompiler{}).Compile(ctx, publication.Input{}, repo, media, writer)
			_, _ = writer.Finalize()
		}
	}

	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	sessionToken := hex.EncodeToString(tokenBytes)

	var app *application.App
	var window *application.WebviewWindow

	onPickFolder := func() (string, error) {
		if app == nil || window == nil {
			return "", fmt.Errorf("desktop window is not initialized")
		}
		return app.Dialog.OpenFile().
			CanChooseDirectories(true).
			CanChooseFiles(false).
			CanCreateDirectories(true).
			SetTitle("Select Workspace Folder").
			AttachToWindow(window).
			PromptForSingleSelection()
	}

	var previewServer *PreviewServer
	if *mode == "admin" {
		readerSub, _ := fs.Sub(embeddedAssets, "assets/reader")
		ps, err := StartPreviewServer("127.0.0.1:8081", func() string { return actualPublic }, readerSub)
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
		Token:     sessionToken,
		PreviewPortFn: func() string {
			if previewServer != nil {
				return previewServer.Port()
			}
			return ""
		},
		OnPickFolder: onPickFolder,
	})
	if err != nil {
		return fmt.Errorf("initialize handler: %w", err)
	}

	app = application.New(application.Options{
		Name:        "Felicia Studio",
		Description: "Map-based travel journal studio",
		Assets: application.AssetOptions{
			Handler: handler,
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyRegular,
		},
		RawMessageHandler: func(_ application.Window, message string, _ *application.OriginInfo) {
			if !strings.HasPrefix(message, "desktop:") || len(message) > 1<<20 {
				return
			}
			var payload struct {
				Token      string          `json:"token"`
				Diagnostic json.RawMessage `json:"diagnostic"`
				Evidence   json.RawMessage `json:"evidence"`
			}
			raw := strings.TrimPrefix(message, "desktop:")
			if err := json.Unmarshal([]byte(raw), &payload); err != nil || payload.Token != sessionToken {
				return
			}
			if len(payload.Evidence) > 0 {
				fmt.Println("NATIVE_EVIDENCE=" + string(payload.Evidence))
				if *smoke {
					time.AfterFunc(300*time.Millisecond, app.Quit)
				}
			}
		},
	})

	title := "Felicia Studio"
	if *mode == "reader" {
		title = "Felicia — Public Reader"
	}

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  title,
		Width:  1100,
		Height: 760,
		URL:    "/",
	})

	if *smoke {
		// In automated smoke runs, ensure timeout kills app if stuck
		timer := time.AfterFunc(60*time.Second, func() {
			fmt.Fprintln(os.Stderr, "felicia-desktop: smoke check timed out")
			app.Quit()
		})
		defer timer.Stop()
	}

	return app.Run()
}

func resolveDefaultWorkspace() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".felicia"), nil
}
