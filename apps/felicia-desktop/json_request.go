package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const desktopJSONLimit = 8 << 20 // Essays and metadata only; binary uploads use their own handlers.

func decodeDesktopJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if r.ContentLength > desktopJSONLimit {
		respondError(w, http.StatusRequestEntityTooLarge, "request JSON exceeds 8 MiB")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, desktopJSONLimit))
	err := decoder.Decode(value)
	if err == nil {
		var extra any
		err = decoder.Decode(&extra)
		if errors.Is(err, io.EOF) {
			return true
		}
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		respondError(w, http.StatusRequestEntityTooLarge, "request JSON exceeds 8 MiB")
	} else {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
	}
	return false
}
