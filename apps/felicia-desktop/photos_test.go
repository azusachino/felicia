package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
)

func TestSamplePhotoCurationKeepsOriginalIdentityAndConfinesReads(t *testing.T) {
	root, err := createSampleWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(root) }()
	repo, err := sqlite.Open(filepath.Join(root, "felicia.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Close() }()
	h, err := NewHandler(HandlerConfig{Repo: repo, MediaRoot: filepath.Join(root, "media"), PublicDir: filepath.Join(root, "site"), Mode: "admin", Sample: true})
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse("0190cbde-f300-7000-8000-000000000020")
	photo, err := repo.GetPhoto(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	contentURL := "/api/admin/photos/" + id.String() + "/content"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, contentURL, nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content: %d %s", w.Code, w.Header())
	}
	caption := "Saved sample caption"
	photo.Caption = &caption
	body, _ := json.Marshal(photo)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/admin/photos", bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("save: %d %s", w.Code, w.Body)
	}
	photo.ObjectKey = "../outside.png"
	body, _ = json.Marshal(photo)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/admin/photos", bytes.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("identity change accepted: %d", w.Code)
	}
	outside := filepath.Join(root, "private-sentinel")
	if err = os.WriteFile(outside, []byte("do not disclose"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside, filepath.Join(root, "media", "escape.png")); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../private-sentinel", "escape.png", outside} {
		photo.ObjectKey = key
		if err = repo.UpsertPhoto(context.Background(), photo); err != nil {
			t.Fatal(err)
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, contentURL, nil))
		if w.Code == http.StatusOK || bytes.Contains(w.Body.Bytes(), []byte("do not disclose")) {
			t.Fatalf("escaped media root through %q: %d %s", key, w.Code, w.Body)
		}
	}
}
