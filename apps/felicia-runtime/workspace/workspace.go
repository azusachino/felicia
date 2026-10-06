// Package workspace defines standard layout and resolution for Felicia workspaces.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

// Workspace describes the canonical directories and database location of a Felicia workspace.
type Workspace struct {
	Root      string
	Database  string
	MediaRoot string
	PublicDir string
}

// New constructs a Workspace anchored at the given root directory.
func New(root string) Workspace {
	cleaned := filepath.Clean(root)
	return Workspace{
		Root:      cleaned,
		Database:  filepath.Join(cleaned, "felicia.sqlite"),
		MediaRoot: filepath.Join(cleaned, "media"),
		PublicDir: filepath.Join(cleaned, "site"),
	}
}

// EnsureDirs creates the workspace root and its managed subdirectories with 0755 permissions.
func (w Workspace) EnsureDirs() error {
	for _, dir := range []string{w.Root, filepath.Dir(w.Database), w.MediaRoot, w.PublicDir} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create workspace directory %s: %w", dir, err)
		}
	}
	return nil
}

// Resolve determines the active workspace following resolution precedence:
// 1. Explicit root (if non-empty)
// 2. FELICIA_WORKSPACE environment variable (if non-empty)
// 3. Local .felicia directory in current working directory (if it exists and is a directory)
// 4. Default ~/.felicia directory in user's home directory.
func Resolve(explicitRoot string) (Workspace, error) {
	if explicitRoot != "" {
		return New(explicitRoot), nil
	}

	if env := os.Getenv("FELICIA_WORKSPACE"); env != "" {
		return New(env), nil
	}

	if cwd, err := os.Getwd(); err == nil {
		localDir := filepath.Join(cwd, ".felicia")
		if info, err := os.Stat(localDir); err == nil && info.IsDir() {
			return New(localDir), nil
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Workspace{}, fmt.Errorf("resolve user home directory: %w", err)
	}

	return New(filepath.Join(homeDir, ".felicia")), nil
}
