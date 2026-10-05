package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestEmbeddedReaderFontMIME(t *testing.T) {
	assets := fstest.MapFS{"fonts/example.woff2": {Data: []byte("wOF2 font bytes")}}
	h := &DesktopHandler{cfg: HandlerConfig{Mode: "reader"}, readerFS: assets}
	w := httptest.NewRecorder()
	h.serveStaticUI(w, httptest.NewRequest(http.MethodGet, "/fonts/example.woff2", nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "font/woff2" || w.Body.String() != "wOF2 font bytes" {
		t.Fatalf("status=%d content-type=%q body=%q", w.Code, w.Header().Get("Content-Type"), w.Body.String())
	}
}
