package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/azusachino/felicia/apps/felicia-publication"
)

// Site & Deploy settings: the directory picker (browse) and persisting a new
// static-output location (ADMIN-02 M2). The browse endpoint is a local-only
// convenience — it lists directories under a fixed root and refuses to escape
// it, so it must never be exposed on a hosted deployment.

type browseEntry struct {
	Name string `json:"name"`
	Path string `json:"path"` // absolute, for selecting as out_dir
}

// handleBrowseDirectories lists the immediate subdirectories of a path under
// the configured browse root. `path` (query) is absolute; empty means the
// root. Requests that resolve outside the root are refused.
func (s *Server) handleBrowseDirectories(w http.ResponseWriter, r *http.Request) {
	root := s.siteBrowseRoot
	if root == "" {
		if home, err := os.UserHomeDir(); err == nil {
			root = home
		} else {
			root = "."
		}
	}
	// Resolve symlinks on the root so all confinement checks compare
	// fully-evaluated paths (filepath.Abs/Rel are lexical and would let a
	// symlink under the root escape it).
	absRoot, err := resolvePath(root)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "browse root is unavailable")
		return
	}

	target := r.URL.Query().Get("path")
	if target == "" {
		target = absRoot
	}
	// Evaluate the target's symlinks too, then confine the *resolved* path:
	// a symlink inside the root that points outside (e.g. link -> /etc) must
	// not slip past a purely lexical check.
	absTarget, err := resolvePath(target)
	if err != nil || !withinRoot(absRoot, absTarget) {
		respondError(w, http.StatusBadRequest, "path is outside the browse root")
		return
	}

	entries, err := os.ReadDir(absTarget)
	if err != nil {
		respondError(w, http.StatusBadRequest, "cannot read directory")
		return
	}
	dirs := []browseEntry{}
	for _, e := range entries {
		if e.IsDir() && e.Name()[0] != '.' {
			// Skip any child that is a symlink escaping the root — the picker
			// must only offer locations genuinely under it.
			child, err := resolvePath(filepath.Join(absTarget, e.Name()))
			if err != nil || !withinRoot(absRoot, child) {
				continue
			}
			dirs = append(dirs, browseEntry{Name: e.Name(), Path: child})
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })

	// parent is empty at the root so the GUI can hide "up".
	parent := ""
	if absTarget != absRoot {
		parent = filepath.Dir(absTarget)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"root":   absRoot,
		"path":   absTarget,
		"parent": parent,
		"dirs":   dirs,
	})
}

// resolvePath returns the absolute, symlink-evaluated form of p. Both steps
// matter for confinement: Abs handles ".." lexically, EvalSymlinks defeats a
// symlink that would otherwise redirect a lexically-valid path outside the
// root. It errors when the path does not exist, which callers treat as "not a
// browsable location under the root".
func resolvePath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// withinRoot reports whether target is root or nested under it. Both arguments
// are expected to be already symlink-resolved (see resolvePath).
func withinRoot(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

type putSiteRequest struct {
	OutDir string `json:"out_dir"`
}

// handlePutSite repoints the static output directory. It updates the running
// server immediately (compile, preview, and build-status all follow) and, when
// a config path is set, persists site.out_dir so it survives a restart.
func (s *Server) handlePutSite(w http.ResponseWriter, r *http.Request) {
	var req putSiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	if req.OutDir == "" {
		respondError(w, http.StatusBadRequest, "out_dir is required")
		return
	}
	// A compile writes artifact files into out_dir and Finalize reconciles it
	// against the artifact manifest. Only an empty directory or a prior felicia
	// site output (identified by its manifest) is a safe target; refuse to
	// repoint output at an unrelated non-empty directory so a stray path can
	// neither be scattered with compiler files nor reconciled unexpectedly.
	if ok, err := outDirIsSafeTarget(req.OutDir); err != nil {
		respondError(w, http.StatusBadRequest, "cannot inspect the output directory")
		return
	} else if !ok {
		respondError(w, http.StatusBadRequest, "output directory is not empty and is not a felicia site output; choose an empty directory or an existing site output")
		return
	}
	if err := os.MkdirAll(req.OutDir, 0o755); err != nil {
		respondError(w, http.StatusBadRequest, "cannot create the output directory")
		return
	}
	s.setSiteOutDir(req.OutDir)
	if err := s.persistSiteOutDir(req.OutDir); err != nil {
		// The session-level change succeeded; report the persistence failure
		// without failing the request so the author can still build now.
		s.logger.WarnContext(r.Context(), "could not persist site out_dir", "err", err, "path", s.configPath)
	}
	respondJSON(w, http.StatusOK, map[string]any{"out_dir": req.OutDir})
}

// outDirIsSafeTarget reports whether dir may be used as the static output
// location. A missing path (created fresh by MkdirAll) and an empty directory
// are safe; a non-empty directory is safe only if it already holds a felicia
// artifact manifest, marking it as a prior site output the compiler owns.
func outDirIsSafeTarget(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	if len(entries) == 0 {
		return true, nil
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(publication.ManifestPath))); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	return false, nil
}

// persistSiteOutDir writes site.out_dir into the TOML config, merging with any
// existing keys. A no-op when no config path is configured.
func (s *Server) persistSiteOutDir(outDir string) error {
	if s.configPath == "" {
		return nil
	}
	values := map[string]any{}
	if raw, err := os.ReadFile(s.configPath); err == nil {
		_ = toml.Unmarshal(raw, &values)
	} else if !os.IsNotExist(err) {
		return err
	}
	site, _ := values["site"].(map[string]any)
	if site == nil {
		site = map[string]any{}
	}
	site["out_dir"] = outDir
	values["site"] = site
	encoded, err := toml.Marshal(values)
	if err != nil {
		return err
	}
	return os.WriteFile(s.configPath, encoded, 0o644)
}
