package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSampleActionsRequirePostAndAnAuthorizedNativeCallback(t *testing.T) {
	opened := 0
	h := &DesktopHandler{cfg: HandlerConfig{Mode: "admin", Token: "test-session", OnOpenSample: func() error { opened++; return nil }}}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(method, "/api/desktop/sample", nil))
		if w.Code == http.StatusOK || opened != 0 {
			t.Fatalf("unauthorized open: method=%s status=%d count=%d", method, w.Code, opened)
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/api/desktop/sample", nil)
	request.Header.Set("X-Desktop-Token", "test-session")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != http.StatusOK || opened != 1 {
		t.Fatalf("open: %d count=%d", w.Code, opened)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/desktop/sample/close", nil)
	request.Header.Set("X-Desktop-Token", "test-session")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("closed a real workspace: %d", w.Code)
	}
}
