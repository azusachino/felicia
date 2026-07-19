package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
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
	absRoot, err := filepath.Abs(root)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "browse root is unavailable")
		return
	}

	target := r.URL.Query().Get("path")
	if target == "" {
		target = absRoot
	}
	absTarget, err := filepath.Abs(target)
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
			dirs = append(dirs, browseEntry{Name: e.Name(), Path: filepath.Join(absTarget, e.Name())})
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

// withinRoot reports whether target is root or nested under it.
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
