package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/azusachino/felicia/apps/felicia-server/api"
)

func TestServerHealthAndReadiness(t *testing.T) {
	server := api.NewServer(newMockRepository(), loadKinds(t), api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{})
	handler := server.Handler()

	for _, test := range []struct {
		path string
		want int
	}{
		{path: "/healthz", want: http.StatusOK},
		{path: "/readyz", want: http.StatusOK},
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if recorder.Code != test.want {
			t.Fatalf("%s: status = %d, want %d", test.path, recorder.Code, test.want)
		}
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("health response is missing security headers")
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("health response is missing request ID")
	}
}

func TestServerRateLimitAndCORS(t *testing.T) {
	server := api.NewServer(newMockRepository(), loadKinds(t), api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{
		RatePerSecond:  0.001,
		RateBurst:      1,
		AllowedOrigin:  "https://admin.example",
		RequestTimeout: time.Second,
	})
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/journeys", nil)
	request.Header.Set("Origin", "https://admin.example")
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, request)
	if first.Code == http.StatusTooManyRequests || first.Header().Get("Access-Control-Allow-Origin") != "https://admin.example" {
		t.Fatalf("first request was not allowed with CORS: %d %q", first.Code, first.Header().Get("Access-Control-Allow-Origin"))
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/v1/journeys", nil))
	if second.Code != http.StatusTooManyRequests || second.Header().Get("Retry-After") != "1" {
		t.Fatalf("second request status = %d, Retry-After = %q", second.Code, second.Header().Get("Retry-After"))
	}

	options := httptest.NewRequest(http.MethodOptions, "/api/v1/journeys", nil)
	options.Header.Set("Origin", "https://admin.example")
	preflight := httptest.NewRecorder()
	handler.ServeHTTP(preflight, options)
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", preflight.Code, http.StatusNoContent)
	}
}
