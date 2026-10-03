//go:build e2e

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestE2EGuard(t *testing.T) {
	const host = "127.0.0.1:12345"
	const token = "synthetic-test-token-only-32bytes"
	h := guardE2E(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), host, token)
	for _, tc := range []struct {
		name, host, origin, token string
		want                      int
	}{
		{"valid", host, "http://" + host, token, 204},
		{"no-origin", host, "", token, 204},
		{"missing-token", host, "", "", 403},
		{"wrong-token", host, "", "wrong", 403},
		{"wrong-host", "evil.example", "", token, 403},
		{"cross-origin", host, "https://evil.example", token, 403},
		{"null-origin", host, "null", token, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "http://"+tc.host+"/api/admin/journeys", nil)
			r.Header.Set("Origin", tc.origin)
			if tc.token != "" {
				r.AddCookie(&http.Cookie{Name: "felicia-e2e", Value: tc.token})
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status=%d, want %d", w.Code, tc.want)
			}
		})
	}
}

func TestE2EConfiguration(t *testing.T) {
	previous := *e2eAddr
	t.Cleanup(func() { *e2eAddr = previous })
	t.Setenv("FELICIA_E2E_TOKEN", strings.Repeat("a", 32))
	for _, addr := range []string{"0.0.0.0:0", "localhost:0", "192.0.2.1:0", "bad"} {
		*e2eAddr = addr
		if err := validateE2EPaths("/synthetic/db", "/synthetic/media", "/synthetic/site"); err == nil {
			t.Fatalf("accepted %q", addr)
		}
	}
	*e2eAddr = "127.0.0.1:0"
	if err := validateE2EPaths("", "/synthetic/media", "/synthetic/site"); err == nil {
		t.Fatal("accepted default private database path")
	}
	if err := validateE2EPaths("/synthetic/db", "/synthetic/media", "/synthetic/site"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FELICIA_E2E_TOKEN", "")
	if err := validateE2EPaths("/synthetic/db", "/synthetic/media", "/synthetic/site"); err == nil {
		t.Fatal("accepted empty token")
	}
}
