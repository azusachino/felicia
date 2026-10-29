package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStudioOrigin(t *testing.T) {
	for _, tc := range []struct {
		url  string
		want bool
	}{
		{"mygo://localhost/", true},
		{"http://mygo.localhost/", true},
		{"https://example.com/", false},
		{"mygo://attacker/", false},
		{"mygo://localhost.evil/", false},
		{"mygo://user@localhost/", false},
		{"null", false},
	} {
		if got := studioOrigin(tc.url); got != tc.want {
			t.Errorf("studioOrigin(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}

func TestDesktopProtocolRejectsRequestsWithoutWebview(t *testing.T) {
	called := false
	h := desktopProtocol(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "mygo://localhost/api/admin/compile", nil))
	if w.Code != http.StatusForbidden || called {
		t.Fatalf("status = %d, handler called = %v", w.Code, called)
	}
}
