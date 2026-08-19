package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/azusachino/felicia/core/domain"
	"github.com/azusachino/felicia/publication"
)

// Site identity & style settings (ADMIN-02 M2): GET/PUT /api/admin/site-settings.
// This is deliberately a separate file and resource from sitesettings.go's
// /api/admin/site, which owns unrelated process configuration (out_dir,
// preview_port) — same-ish name, different concept; do not merge them.

// accentPattern is the bare hex format the admin UI's <input type="color">
// already emits ("#rrggbb"); empty means "unset".
var accentPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// handleGetSiteSettings resolves the sole journal's site identity/style
// settings, defaulting (200, not 404) when none have been saved yet.
func (s *Server) handleGetSiteSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := publication.ResolveSiteSettings(r.Context(), s.repo)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, publication.NewStaticSiteSettings(settings))
}

// putSiteSettingsRequest is a partial update: only non-nil fields are
// applied onto the current (or default) settings, so the admin GUI's single
// "Save site settings" action can send just the fields it edited.
type putSiteSettingsRequest struct {
	Title           *string `json:"title"`
	Description     *string `json:"description"`
	Design          *string `json:"design"`
	DefaultLanguage *string `json:"default_language"`
	DefaultTheme    *string `json:"default_theme"`
	Accent          *string `json:"accent"`
}

// handlePutSiteSettings applies a partial update to the site identity/style
// settings and invalidates the public cache so the live endpoint and the
// next static compile both reflect it immediately. No optimistic
// concurrency: single-row, low-contention, out of this milestone's scope.
func (s *Server) handlePutSiteSettings(w http.ResponseWriter, r *http.Request) {
	var req putSiteSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}

	var issues []domain.Issue
	if req.Design != nil && !validSiteDesign(*req.Design) {
		issues = append(issues, domain.Issue{Field: "design", Code: "invalid_enum"})
	}
	if req.DefaultLanguage != nil && !validSiteLanguage(*req.DefaultLanguage) {
		issues = append(issues, domain.Issue{Field: "default_language", Code: "invalid_enum"})
	}
	if req.DefaultTheme != nil && !validSiteTheme(*req.DefaultTheme) {
		issues = append(issues, domain.Issue{Field: "default_theme", Code: "invalid_enum"})
	}
	if req.Accent != nil && *req.Accent != "" && !accentPattern.MatchString(*req.Accent) {
		issues = append(issues, domain.Issue{Field: "accent", Code: "invalid_format"})
	}
	if len(issues) > 0 {
		respondJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":  "validation failed",
			"issues": issues,
		})
		return
	}

	current, err := publication.ResolveSiteSettings(r.Context(), s.repo)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.Title != nil {
		current.Title = *req.Title
	}
	if req.Description != nil {
		current.Description = *req.Description
	}
	if req.Design != nil {
		current.Design = *req.Design
	}
	if req.DefaultLanguage != nil {
		current.DefaultLanguage = *req.DefaultLanguage
	}
	if req.DefaultTheme != nil {
		current.DefaultTheme = *req.DefaultTheme
	}
	if req.Accent != nil {
		current.Accent = *req.Accent
	}

	if err := s.repo.UpsertSiteSettings(r.Context(), &current); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.cache.InvalidateAll(r.Context())
	respondJSON(w, http.StatusOK, publication.NewStaticSiteSettings(current))
}

func validSiteDesign(value string) bool {
	switch value {
	case "v1", "v2", "v3", "v4":
		return true
	default:
		return false
	}
}

func validSiteLanguage(value string) bool {
	switch value {
	case "ja", "en", "zh":
		return true
	default:
		return false
	}
}

func validSiteTheme(value string) bool {
	switch value {
	case "dark", "light":
		return true
	default:
		return false
	}
}
