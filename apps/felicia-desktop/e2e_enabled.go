//go:build e2e

package main

import (
	"crypto/subtle"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var e2eAddr = flag.String("e2e-addr", "", "test-build-only authenticated loopback host:port")

func validateE2EPaths(db, media, public string) error {
	if *e2eAddr == "" {
		return nil
	}
	for _, p := range []string{db, media, public} {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("e2e requires explicit absolute database, media and output paths")
		}
	}
	host, _, err := net.SplitHostPort(*e2eAddr)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("e2e address must be an IP-literal loopback host:port")
	}
	if len(os.Getenv("FELICIA_E2E_TOKEN")) < 32 {
		return fmt.Errorf("e2e requires a run token of at least 32 bytes")
	}
	return nil
}

func guardE2E(next http.Handler, host, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("felicia-e2e")
		if r.Host != host || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+host) ||
			err != nil || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(token)) != 1 {
			http.Error(w, "e2e request rejected", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// Reuses the real handler/assets/repository composition, not the native webview.
func serveE2E(handler http.Handler) (bool, error) {
	if *e2eAddr == "" {
		return false, nil
	}
	ln, err := net.Listen("tcp", *e2eAddr)
	if err != nil {
		return true, fmt.Errorf("start e2e listener: %w", err)
	}
	defer func() { _ = ln.Close() }()
	host := ln.Addr().String()
	server := &http.Server{
		Handler:           guardE2E(handler, host, os.Getenv("FELICIA_E2E_TOKEN")),
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Printf("FELICIA_E2E_READY=http://%s\n", host)
	return true, server.Serve(ln)
}
