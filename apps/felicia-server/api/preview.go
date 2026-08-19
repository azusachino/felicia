package api

import (
	"net/http"
	"os"
	"path"
	"path/filepath"

	publication "github.com/azusachino/felicia/apps/felicia-publication"
)

// handleSiteInfo describes the local build-and-preview setup so the GUI's
// Site page can render the build target, the preview link, and a hint when
// the pre-built public SPA is missing.
func (s *Server) handleSiteInfo(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"out_dir":        s.SiteOutDir(),
		"preview_port":   s.sitePreviewPort,
		"spa_ready":      fileExists(filepath.Join(s.siteSpaDist, "index.html")),
		"artifact_ready": fileExists(filepath.Join(s.SiteOutDir(), filepath.FromSlash(publication.ManifestPath))),
	})
}

// PreviewHandler serves the compiled site the way a static host would: the
// compile output (api/v1 JSON + safe media) overlaid on a pre-built public
// SPA. Artifact files win; everything else (index.html, JS/CSS assets) falls
// through to the SPA dist. outDir is a getter so the preview follows a
// runtime change to the Site & Deploy output location. Both roots are
// read-only.
func PreviewHandler(outDir func() string, spaDist string) http.Handler {
	spa := http.FileServer(http.Dir(spaDist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		root := outDir()
		clean := path.Clean("/" + r.URL.Path)
		if clean != "/" && fileExists(filepath.Join(root, filepath.FromSlash(clean))) {
			http.FileServer(http.Dir(root)).ServeHTTP(w, r)
			return
		}
		spa.ServeHTTP(w, r)
	})
}

func fileExists(name string) bool {
	info, err := os.Stat(name)
	return err == nil && !info.IsDir()
}
