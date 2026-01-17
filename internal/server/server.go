package server

import (
	"net/http"

	"github.com/LealKevin/keiko/internal/store"
)

type Server struct {
	store *store.Store
}

func New(store *store.Store) *Server {
	return &Server{store: store}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Apply rate limiting to API routes
	rateLimiter := NewRateLimiter(30, 60) // 30 requests per 60 seconds

	// Health check (no rate limiting)
	mux.HandleFunc("GET /health", s.handleHealth)

	// API v1 routes (with rate limiting)
	mux.Handle("GET /api/v1/news", rateLimiter.Middleware(http.HandlerFunc(s.handleGetNews)))
	mux.Handle("GET /api/v1/news/{id}", rateLimiter.Middleware(http.HandlerFunc(s.handleGetNewsById)))

	// CORS middleware
	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
