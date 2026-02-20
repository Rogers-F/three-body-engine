package ipc

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

// Server wraps an HTTP server with engine-specific routing.
type Server struct {
	httpServer *http.Server
}

// NewServer creates a Server that binds to the given address.
// If a dist/ directory exists next to the executable (or in cwd),
// it serves the frontend UI at "/" and auto-opens the browser.
func NewServer(h *Handler, listenAddr string) *Server {
	mux := http.NewServeMux()

	// Health endpoint.
	mux.HandleFunc("GET /api/v1/health", h.Health)

	// Flow endpoints.
	mux.HandleFunc("POST /api/v1/flow", h.CreateFlow)
	mux.HandleFunc("GET /api/v1/flow/{taskID}", h.GetFlow)
	mux.HandleFunc("POST /api/v1/flow/{taskID}/advance", h.AdvanceFlow)

	// Worker endpoint.
	mux.HandleFunc("GET /api/v1/flow/{taskID}/workers", h.ListWorkers)

	// Event endpoints.
	mux.HandleFunc("GET /api/v1/flow/{taskID}/events", h.ListEvents)
	mux.HandleFunc("GET /api/v1/flow/{taskID}/events/stream", h.StreamEvents)

	// Review endpoint.
	mux.HandleFunc("GET /api/v1/flow/{taskID}/reviews", h.ListReviews)

	// Cost endpoint.
	mux.HandleFunc("GET /api/v1/flow/{taskID}/cost", h.GetCost)

	// Serve frontend static files if dist/ directory exists.
	if distDir := findDistDir(); distDir != "" {
		log.Printf("serving frontend from %s", distDir)
		fs := http.FileServer(spaFileSystem{root: http.Dir(distDir)})
		mux.Handle("/", fs)
	}

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: corsMiddleware(mux),
	}

	return &Server{
		httpServer: srv,
	}
}

// Start begins listening for HTTP connections. Blocks until the server stops.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// corsMiddleware adds CORS headers for local desktop app access.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// findDistDir looks for a dist/ directory next to the executable, then in cwd.
func findDistDir() string {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "dist")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	if info, err := os.Stat("dist"); err == nil && info.IsDir() {
		return "dist"
	}
	return ""
}

// spaFileSystem wraps http.Dir to serve index.html for unknown paths (SPA routing).
type spaFileSystem struct {
	root http.FileSystem
}

func (s spaFileSystem) Open(name string) (http.File, error) {
	f, err := s.root.Open(name)
	if err != nil {
		// If file not found, serve index.html for client-side routing.
		return s.root.Open("/index.html")
	}

	// If it's a directory, check for index.html inside it.
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return s.root.Open("/index.html")
	}
	if stat.IsDir() {
		indexPath := path.Join(name, "index.html")
		if _, err := s.root.Open(indexPath); err != nil {
			f.Close()
			return s.root.Open("/index.html")
		}
	}

	return f, nil
}

// FormatListenURL returns a clickable URL for the listen address.
func FormatListenURL(addr string) string {
	if addr[0] == ':' {
		return fmt.Sprintf("http://localhost%s", addr)
	}
	return fmt.Sprintf("http://%s", addr)
}
