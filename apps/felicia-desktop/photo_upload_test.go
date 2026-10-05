package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	photoruntime "github.com/azusachino/felicia/apps/felicia-runtime/photos"
)

const uploadMementoID = "0190cbde-f300-7000-8000-000000000010"

func photoUploadHandler(t *testing.T, isolated bool) (*DesktopHandler, *sqlite.Repository, string) {
	t.Helper()
	root, err := createSampleWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	repo, err := sqlite.Open(filepath.Join(root, "felicia.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	h, err := NewHandler(HandlerConfig{Repo: repo, MediaRoot: filepath.Join(root, "media"), PublicDir: filepath.Join(root, "site"), Mode: "admin", Sample: isolated, Isolated: isolated,
		OnPickFile: func(string) (string, error) { t.Fatal("photo API invoked native picker"); return "", nil }})
	if err != nil {
		t.Fatal(err)
	}
	return h, repo, root
}

func uploadPhotoRequest(t *testing.T, id, filename string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/admin/mementos/"+id+"/photos/upload", &body)
	r.Header.Set("Content-Type", form.FormDataContentType())
	return r
}

func TestDesktopPhotoUploadStoresPrivateBytesWithoutSourcePaths(t *testing.T) {
	h, repo, root := photoUploadHandler(t, false)
	var original bytes.Buffer
	if err := jpeg.Encode(&original, image.NewGray(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, uploadPhotoRequest(t, uploadMementoID, "/private/sensitive-source.jpg", original.Bytes()))
	if w.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "sensitive-source") || strings.Contains(w.Body.String(), root) {
		t.Fatal("source path leaked")
	}
	var photo domain.MementoPhoto
	if err := json.Unmarshal(w.Body.Bytes(), &photo); err != nil {
		t.Fatal(err)
	}
	if photo.Seq != 2 || !strings.HasPrefix(photo.ContentHash, "sha256:") || !strings.HasSuffix(photo.ObjectKey, "/original.jpg") || photo.SourceRef != nil {
		t.Fatalf("identity: %+v", photo)
	}
	data, err := os.ReadFile(filepath.Join(root, "media", filepath.FromSlash(photo.ObjectKey)))
	if err != nil || !bytes.Equal(data, original.Bytes()) {
		t.Fatalf("private original mismatch: %v", err)
	}
	content := httptest.NewRecorder()
	h.ServeHTTP(content, httptest.NewRequest(http.MethodGet, "/api/admin/photos/"+photo.ID.String()+"/content", nil))
	if content.Code != http.StatusOK || !bytes.Equal(content.Body.Bytes(), original.Bytes()) {
		t.Fatalf("content: %d", content.Code)
	}
	stored, err := repo.GetPhoto(context.Background(), photo.ID)
	if err != nil || stored.ObjectKey != photo.ObjectKey {
		t.Fatalf("photo missing: %v", err)
	}

	for name, data := range map[string][]byte{"not-an-image.jpg": []byte("<script/>"), "phone.heic": []byte("\x00\x00\x00\x18ftypheic"), "huge.jpg": make([]byte, photoruntime.MaxUploadBytes+1)} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			h.ServeHTTP(response, uploadPhotoRequest(t, uploadMementoID, name, data))
			want := http.StatusUnsupportedMediaType
			if name == "huge.jpg" {
				want = http.StatusRequestEntityTooLarge
			}
			if response.Code != want {
				t.Fatalf("rejection: %d %s", response.Code, response.Body)
			}
		})
	}
	photos, err := repo.ListPhotosByMemento(context.Background(), uuid.MustParse(uploadMementoID))
	if err != nil || len(photos) != 3 {
		t.Fatalf("rejected input created photo: %d %v", len(photos), err)
	}
}

func TestTemporaryPhotoUploadIsGeneratedOnly(t *testing.T) {
	h, repo, _ := photoUploadHandler(t, true)
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/admin/browse?path=/private", nil),
		httptest.NewRequest(http.MethodPut, "/api/admin/site", strings.NewReader(`{"out_dir":"/private"}`)),
	} {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("isolated output access: %d %s", response.Code, response.Body)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, uploadPhotoRequest(t, uploadMementoID, "real-file.jpg", []byte("never open files")))
	if w.Code != http.StatusForbidden {
		t.Fatalf("temporary external upload: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/admin/mementos/"+uploadMementoID+"/photos/sample", nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("generated upload: %d %s", w.Code, w.Body)
	}
	var photo domain.MementoPhoto
	if err := json.Unmarshal(w.Body.Bytes(), &photo); err != nil {
		t.Fatal(err)
	}
	if photo.Seq != 2 || photo.MementoID.String() != uploadMementoID {
		t.Fatalf("generated identity: %+v", photo)
	}
	rows, err := repo.ListPhotosByMemento(context.Background(), uuid.MustParse(uploadMementoID))
	if err != nil || len(rows) != 3 {
		t.Fatalf("generated photo missing: %v", err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/admin/mementos/"+uploadMementoID+"/photos/sample", strings.NewReader("input")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("generated endpoint accepted input: %d", w.Code)
	}
}
