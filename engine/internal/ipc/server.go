package ipc

import (
	"context"
	"net/http"
)

// Server wraps an HTTP server with engine-specific routing.
type Server struct {
	httpServer *http.Server
}

// NewServer creates a Server that binds to the given address.
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
