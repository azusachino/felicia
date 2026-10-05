package main

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
)

// Each workspace retains its own immutable handler and resources. Switching
// waits for in-flight authoring requests; a request never changes repositories
// halfway through its execution. No child process or additional window exists.
type WorkspaceRouter struct {
	mu            sync.RWMutex
	original      *DesktopHandler
	current       *DesktopHandler
	sampleRepo    *sqlite.Repository
	samplePreview *PreviewServer
	sampleRoot    string
	onChange      func(bool)
}

func NewWorkspaceRouter(original *DesktopHandler, onChange func(bool)) *WorkspaceRouter {
	router := &WorkspaceRouter{original: original, current: original, onChange: onChange}
	original.cfg.OnOpenSample = router.openSample
	return router
}

func (s *WorkspaceRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The transition callback runs synchronously under exclusive ownership.
	cleanPath := path.Clean(r.URL.Path)
	if cleanPath == "/api/desktop/sample" || cleanPath == "/api/desktop/sample/close" {
		s.mu.Lock()
		defer s.mu.Unlock()
	} else {
		s.mu.RLock()
		defer s.mu.RUnlock()
	}
	s.current.ServeHTTP(w, r)
}

func (s *WorkspaceRouter) openSample() (err error) {
	if s.current != s.original || s.original.cfg.Sample {
		return fmt.Errorf("sample workspace is already open")
	}
	root, err := createSampleWorkspace()
	if err != nil {
		return err
	}
	repo, err := sqlite.Open(filepath.Join(root, "felicia.sqlite"))
	if err != nil {
		_ = os.RemoveAll(root)
		return err
	}
	reader, err := fs.Sub(embeddedAssets, "assets/reader")
	if err != nil {
		_ = repo.Close()
		_ = os.RemoveAll(root)
		return err
	}
	preview, err := StartPreviewServer("127.0.0.1:0", func() string { return filepath.Join(root, "site") }, reader)
	if err != nil {
		_ = repo.Close()
		_ = os.RemoveAll(root)
		return err
	}
	defer func() {
		if err != nil {
			_ = preview.Close()
			_ = repo.Close()
			_ = os.RemoveAll(root)
		}
	}()
	cfg := s.original.cfg
	cfg.Repo, cfg.MediaRoot, cfg.PublicDir = repo, filepath.Join(root, "media"), filepath.Join(root, "site")
	cfg.Sample, cfg.ReturnAvailable = true, !s.original.cfg.Isolated
	cfg.PreviewPortFn = preview.Port
	cfg.OnCloseSample = func() error {
		s.current = s.original
		s.closeSampleResources()
		if s.onChange != nil {
			s.onChange(false)
		}
		return nil
	}
	handler, err := NewHandler(cfg)
	if err != nil {
		return err
	}
	s.sampleRepo, s.samplePreview, s.sampleRoot = repo, preview, root
	s.current = handler
	if s.onChange != nil {
		s.onChange(true)
	}
	return nil
}

func (s *WorkspaceRouter) closeSampleResources() {
	if s.samplePreview != nil {
		_ = s.samplePreview.Close()
		s.samplePreview = nil
	}
	if s.sampleRepo != nil {
		_ = s.sampleRepo.Close()
		s.sampleRepo = nil
	}
	if s.sampleRoot != "" {
		_ = os.RemoveAll(s.sampleRoot)
		s.sampleRoot = ""
	}
}

func (s *WorkspaceRouter) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = s.original
	s.closeSampleResources()
}
