package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/egoist/mygo"
)

// desktopProtocol confines privileged routes to the studio page and origin.
// A webview scheme is not a TCP listener, but other pages can still issue
// cross-origin requests to it even when they cannot read the response.
func desktopProtocol(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			page := mygo.CallerPage(r.Context())
			if page == nil || !studioOrigin(page.URL()) || (r.Header.Get("Origin") != "" && !studioOrigin(r.Header.Get("Origin"))) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		handler.ServeHTTP(w, r)
	})
}

func studioOrigin(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.User == nil && ((u.Scheme == "mygo" && u.Host == "localhost") || (u.Scheme == "http" && u.Host == "mygo.localhost"))
}
