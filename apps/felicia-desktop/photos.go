package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

func (h *DesktopHandler) handleMementoPhotos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := uuid.Parse(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/admin/mementos/"), "/photos"))
	if err != nil {
		http.Error(w, "invalid memento id", http.StatusBadRequest)
		return
	}
	if _, err = h.cfg.Repo.GetMemento(r.Context(), id); err != nil {
		http.Error(w, "memento not found", http.StatusNotFound)
		return
	}
	photos, err := h.cfg.Repo.ListPhotosByMemento(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if photos == nil {
		photos = []*domain.MementoPhoto{}
	}
	respondJSON(w, http.StatusOK, photos)
}

// Curation changes metadata only. Existing originals cannot be replaced or
// pointed outside the private media store through this endpoint.
func (h *DesktopHandler) handleUpdatePhoto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var requested domain.MementoPhoto
	if !decodeDesktopJSON(w, r, &requested) {
		return
	}
	if requested.Seq < 0 {
		http.Error(w, "invalid photo metadata", http.StatusBadRequest)
		return
	}
	stored, err := h.cfg.Repo.GetPhoto(r.Context(), requested.ID)
	if err != nil {
		http.Error(w, "photo not found", http.StatusNotFound)
		return
	}
	if stored.MementoID != requested.MementoID || stored.ObjectKey != requested.ObjectKey || stored.ContentHash != requested.ContentHash {
		http.Error(w, "original photo identity cannot change", http.StatusBadRequest)
		return
	}
	stored.Caption, stored.Seq, stored.TakenAt, stored.SourceRef = requested.Caption, requested.Seq, requested.TakenAt, requested.SourceRef
	if err = h.cfg.Repo.UpsertPhoto(r.Context(), stored); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *DesktopHandler) handlePhotoContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := uuid.Parse(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/admin/photos/"), "/content"))
	if err != nil {
		http.Error(w, "invalid photo id", http.StatusBadRequest)
		return
	}
	photo, err := h.cfg.Repo.GetPhoto(r.Context(), id)
	if err != nil {
		http.Error(w, "photo not found", http.StatusNotFound)
		return
	}
	if !filepath.IsLocal(photo.ObjectKey) {
		http.Error(w, "invalid original path", http.StatusBadRequest)
		return
	}
	root, err := os.OpenRoot(h.cfg.MediaRoot)
	if err != nil {
		http.Error(w, "media unavailable", http.StatusNotFound)
		return
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(photo.ObjectKey)
	if err != nil {
		http.Error(w, "original not found", http.StatusNotFound)
		return
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		http.Error(w, "original unavailable", http.StatusNotFound)
		return
	}
	http.ServeContent(w, r, photo.ObjectKey, info.ModTime(), file)
}
