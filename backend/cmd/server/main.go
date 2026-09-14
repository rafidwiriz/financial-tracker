package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rafidwiriz/financial-tracker/backend/internal/config"
	"github.com/rafidwiriz/financial-tracker/backend/internal/database"
	"github.com/rafidwiriz/financial-tracker/backend/internal/repository"
	"github.com/rafidwiriz/financial-tracker/backend/internal/server"
)

// Usage:
//
//	server serve    (default) — migrate, seed the default user, serve HTTP
//	server migrate           — run migrations and exit (D-014)
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}
	setupLogger(cfg.LogLevel)

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := database.Migrate(cfg.DatabaseURL); err != nil {
			slog.Error("migrate failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations applied")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Refuse to serve on migration failure (D-014).
	if err := database.Migrate(cfg.DatabaseURL); err != nil {
		slog.Error("migrate failed, refusing to start", "error", err)
		os.Exit(1)
	}
	if err := repository.NewUserRepo(pool).EnsureDefaultUser(ctx, cfg.DefaultUserID); err != nil {
		slog.Error("seed default user failed", "error", err)
		os.Exit(1)
	}

	srv := server.New(cfg, pool)

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server exited", "error", err)
			stop()
		}
	}()

	<-runCtx.Done()
	slog.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

func setupLogger(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}