package main

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// PreviewServer serves the compiled static site overlaid on the embedded
// reader SPA. The compiled artifacts (api/v1 JSON and media) take precedence;
// anything else falls back to the embedded reader UI.
type PreviewServer struct {
	listener net.Listener
	server   *http.Server
	outDirFn func() string
	readerFS fs.FS
	port     string
	mu       sync.RWMutex
	running  bool
}

func StartPreviewServer(preferredAddr string, outDirFn func() string, readerFS fs.FS) (*PreviewServer, error) {
	if preferredAddr == "" {
		preferredAddr = "127.0.0.1:8081"
	}

	ln, err := net.Listen("tcp", preferredAddr)
	if err != nil {
		// If preferred port is in use, bind ephemeral loopback port
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, fmt.Errorf("listen preview server: %w", err)
		}
	}

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("get preview port: %w", err)
	}

	ps := &PreviewServer{
		listener: ln,
		outDirFn: outDirFn,
		readerFS: readerFS,
		port:     port,
		running:  true,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		clean := path.Clean("/" + r.URL.Path)
		outDir := ps.outDirFn()

		// 1. Try outDir compiled artifact file (e.g. /api/v1/journeys.json or /media/...)
		if clean != "/" {
			artifactPath := filepath.Join(outDir, filepath.FromSlash(clean))
			if info, err := os.Stat(artifactPath); err == nil && !info.IsDir() {
				http.FileServer(http.Dir(outDir)).ServeHTTP(w, r)
				return
			}
		}

		// 2. Serve from embedded reader filesystem
		assetPath := strings.TrimPrefix(clean, "/")
		if assetPath == "" {
			assetPath = "index.html"
		}
		data, err := fs.ReadFile(ps.readerFS, assetPath)
		if err != nil {
			// SPA fallback: return index.html for client-side routes
			indexData, indexErr := fs.ReadFile(ps.readerFS, "index.html")
			if indexErr != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(indexData)
			return
		}

		ctype := "application/octet-stream"
		switch filepath.Ext(assetPath) {
		case ".html":
			ctype = "text/html; charset=utf-8"
		case ".js", ".mjs":
			ctype = "text/javascript; charset=utf-8"
		case ".css":
			ctype = "text/css; charset=utf-8"
		case ".json":
			ctype = "application/json"
		case ".svg":
			ctype = "image/svg+xml"
		case ".png":
			ctype = "image/png"
		}
		w.Header().Set("Content-Type", ctype)
		_, _ = w.Write(data)
	})

	ps.server = &http.Server{
		Handler: handler,
	}

	go func() {
		_ = ps.server.Serve(ln)
	}()

	return ps, nil
}

func (ps *PreviewServer) Port() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.port
}

func (ps *PreviewServer) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if !ps.running {
		return nil
	}
	ps.running = false
	return ps.server.Close()
}
