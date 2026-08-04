package api

import (
	"encoding/json"
	"net/http"

	"github.com/azusachino/felicia/publication"
)

// handleGetPublicSite serves the public site identity/style projection
// (title, description, design, default language/theme, accent). It shares
// publication.ResolveSiteSettings with the static compiler so the live
// endpoint and the compiled api/v1/site.json artifact never diverge on how
// "absent settings" is interpreted (ADMIN-02 M2).
func (s *Server) handleGetPublicSite(w http.ResponseWriter, r *http.Request) {
	cacheKey := "felicia:public:site"
	if cached, err := s.cache.Get(r.Context(), cacheKey); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cached))
		return
	}

	settings, err := publication.ResolveSiteSettings(r.Context(), s.repo)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	projection := publication.NewStaticSiteSettings(settings)

	jsonData, err := json.Marshal(projection)
	if err == nil {
		_ = s.cache.Set(r.Context(), cacheKey, string(jsonData))
	}

	respondJSON(w, http.StatusOK, projection)
}
