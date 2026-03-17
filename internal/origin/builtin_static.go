package origin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// BuiltinStatic serves a static directory over HTTP.
type BuiltinStatic struct {
	cfg      Config
	server   *http.Server
	listener net.Listener
}

// NewBuiltinStatic creates a static file server origin.
func NewBuiltinStatic(cfg Config) (*BuiltinStatic, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("--path is required for builtin:static origin")
	}

	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("checking path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", absPath)
	}

	cfg.Path = absPath
	return &BuiltinStatic{cfg: cfg}, nil
}

func (s *BuiltinStatic) Type() Type {
	return TypeBuiltinStatic
}

func (s *BuiltinStatic) Start(_ context.Context) (string, error) {
	// Bind to a random loopback port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("binding to loopback: %w", err)
	}
	s.listener = listener

	var handler http.Handler
	fs := http.FileServer(http.Dir(s.cfg.Path))

	if s.cfg.SPA {
		// SPA mode: serve index.html for non-file routes.
		index := s.cfg.Index
		if index == "" {
			index = "index.html"
		}
		handler = spaHandler(s.cfg.Path, index, fs)
	} else {
		handler = fs
	}

	// Optionally set Cache-Control.
	if s.cfg.CacheControl != "" {
		handler = cacheControlMiddleware(s.cfg.CacheControl, handler)
	}

	s.server = &http.Server{Handler: handler}

	go s.server.Serve(listener)

	addr := listener.Addr().String()
	return fmt.Sprintf("http://%s", addr), nil
}

func (s *BuiltinStatic) Stop(ctx context.Context) error {
	if s.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
	return nil
}

func (s *BuiltinStatic) Logs() io.ReadCloser {
	return nil
}

func (s *BuiltinStatic) Healthy(_ context.Context) error {
	if s.listener == nil {
		return fmt.Errorf("server not started")
	}
	return nil
}

// spaHandler serves the index file for any path that doesn't match a real file.
func spaHandler(root, index string, fileServer http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(root, filepath.Clean(r.URL.Path))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(root, index))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// cacheControlMiddleware adds a Cache-Control header to all responses.
func cacheControlMiddleware(value string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", value)
		next.ServeHTTP(w, r)
	})
}
