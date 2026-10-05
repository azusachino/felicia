package main

import (
	"bytes"
	"database/sql"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	photoruntime "github.com/azusachino/felicia/apps/felicia-runtime/photos"
)

func photoError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func (h *DesktopHandler) handleUploadPhoto(w http.ResponseWriter, r *http.Request, synthetic bool) {
	if r.Method != http.MethodPost {
		photoError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if synthetic != (h.cfg.Sample || h.cfg.Isolated) {
		photoError(w, http.StatusForbidden, "temporary workspaces accept generated sample photos only")
		return
	}
	suffix := "/photos/upload"
	if synthetic {
		suffix = "/photos/sample"
	}
	id, err := uuid.Parse(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/admin/mementos/"), suffix))
	if err != nil {
		photoError(w, http.StatusBadRequest, "invalid memento id")
		return
	}
	var data []byte
	if synthetic {
		if r.ContentLength != 0 {
			photoError(w, http.StatusBadRequest, "sample photos are generated without file input")
			return
		}
		picture := image.NewRGBA(image.Rect(0, 0, 320, 240))
		draw.Draw(picture, picture.Bounds(), &image.Uniform{color.RGBA{225, 231, 227, 255}}, image.Point{}, draw.Src)
		draw.Draw(picture, image.Rect(60, 60, 260, 180), &image.Uniform{color.RGBA{190, 123, 72, 255}}, image.Point{}, draw.Src)
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, picture); err != nil {
			photoError(w, http.StatusInternalServerError, "could not generate sample photo")
			return
		}
		data = encoded.Bytes()
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, photoruntime.MaxUploadBytes+(1<<20))
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				photoError(w, http.StatusRequestEntityTooLarge, photoruntime.ErrTooLarge.Error())
			} else {
				photoError(w, http.StatusBadRequest, "expected multipart form with a file field")
			}
			return
		}
		if r.MultipartForm != nil {
			defer func() { _ = r.MultipartForm.RemoveAll() }()
		}
		file, _, err := r.FormFile("file") // Never persist the client filename or source path.
		if err != nil {
			photoError(w, http.StatusBadRequest, "photo file is required")
			return
		}
		defer func() { _ = file.Close() }()
		data, err = io.ReadAll(io.LimitReader(file, photoruntime.MaxUploadBytes+1))
		if err != nil {
			photoError(w, http.StatusBadRequest, "could not read uploaded photo")
			return
		}
	}
	photo, err := h.photoWriter.Upload(r.Context(), id, data)
	if err != nil {
		var invalid *photoruntime.InvalidImageError
		switch {
		case errors.Is(err, photoruntime.ErrTooLarge):
			photoError(w, http.StatusRequestEntityTooLarge, err.Error())
		case errors.As(err, &invalid):
			photoError(w, http.StatusUnsupportedMediaType, err.Error())
		case errors.Is(err, domain.ErrNotFound), errors.Is(err, sql.ErrNoRows):
			photoError(w, http.StatusNotFound, "memento not found")
		default:
			photoError(w, http.StatusInternalServerError, "could not store photo")
		}
		return
	}
	respondJSON(w, http.StatusCreated, photo)
}
