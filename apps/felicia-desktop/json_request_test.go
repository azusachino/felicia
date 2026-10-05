package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDesktopJSONBoundariesRejectBeforeRepositoryAccess(t *testing.T) {
	h := &DesktopHandler{} // No repository: a rejected request must never reach a write.
	handlers := map[string]http.HandlerFunc{
		"journey": h.handleUpsertJourney,
		"memento": h.handleUpsertMemento,
		"photo":   h.handleUpdatePhoto,
	}
	oversized := `{"title":"` + strings.Repeat("x", desktopJSONLimit) + `"}`
	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			for _, length := range []int64{int64(len(oversized)), -1} {
				r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(oversized))
				r.ContentLength = length
				w := httptest.NewRecorder()
				handler(w, r)
				if w.Code != http.StatusRequestEntityTooLarge {
					t.Fatalf("length=%d: status=%d body=%s", length, w.Code, w.Body.String())
				}
			}
			w := httptest.NewRecorder()
			handler(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{} {}`)))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("multiple JSON objects: status=%d", w.Code)
			}
		})
	}
}

func TestDesktopJSONAcceptsOneObjectAndWhitespace(t *testing.T) {
	w := httptest.NewRecorder()
	var value struct {
		Title string `json:"title"`
	}
	if !decodeDesktopJSON(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{\"title\":\"Story\"}\n ")), &value) || value.Title != "Story" {
		t.Fatalf("valid JSON rejected: status=%d title=%q", w.Code, value.Title)
	}
}
