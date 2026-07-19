package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/azusachino/felicia/server/api"
)

func TestPreviewHandlerServesArtifactOverSPA(t *testing.T) {
	outDir := t.TempDir()
	spaDist := t.TempDir()

	mustWrite(t, filepath.Join(spaDist, "index.html"), "<html>spa</html>")
	mustWrite(t, filepath.Join(spaDist, "assets", "app.js"), "console.log(1)")
	mustWrite(t, filepath.Join(outDir, "api", "v1", "journeys.json"), "[]")

	handler := api.PreviewHandler(outDir, spaDist)

	cases := []struct {
		path string
		want string
	}{
		{"/", "<html>spa</html>"},
		{"/assets/app.js", "console.log(1)"},
		{"/api/v1/journeys.json", "[]"},
	}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", tc.path, recorder.Code)
		}
		if got := recorder.Body.String(); got != tc.want {
			t.Fatalf("%s: body = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestPreviewHandlerRejectsTraversal(t *testing.T) {
	outDir := t.TempDir()
	spaDist := t.TempDir()
	mustWrite(t, filepath.Join(spaDist, "index.html"), "spa")

	handler := api.PreviewHandler(outDir, spaDist)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.URL.Path = "/../../etc/passwd"
	handler.ServeHTTP(recorder, request)
	if recorder.Code == http.StatusOK && recorder.Body.Len() > 0 && recorder.Body.String() != "spa" {
		t.Fatalf("traversal request returned unexpected content: %q", recorder.Body.String())
	}
}

func mustWrite(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
