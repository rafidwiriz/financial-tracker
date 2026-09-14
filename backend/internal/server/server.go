package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafidwiriz/financial-tracker/backend/internal/config"
	"github.com/rafidwiriz/financial-tracker/backend/internal/server/middleware"
)

// New wires the chi router with middleware (docs/conventions.md §Middleware
// order) and routes, and returns an *http.Server ready to serve.
func New(cfg config.Config, pool *pgxpool.Pool) *http.Server {
	s := &server{cfg: cfg, pool: pool}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	r.Get("/healthz", s.handleHealth)

	return &http.Server{Addr: cfg.HTTPAddr, Handler: r}
}

type server struct {
	cfg  config.Config
	pool *pgxpool.Pool
}

// handleHealth reports server + database health (liveness/probe endpoint —
// intentionally outside /api/v1, no auth, see docs/api-design.md).
func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.pool.Ping(ctx); err != nil {
		slog.Warn("healthz: db unreachable", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}