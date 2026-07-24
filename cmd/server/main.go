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

	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/config"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/server"
)

func main() {
	// Initialize default structured logger outputting JSON or Text to stdout
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Initialize router & server
	srv := server.NewServer(cfg)
	httpServer := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: srv.RegisterRoutes(),
	}

	// 3. Create a context that listens for interrupt signals (Ctrl+C, SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Start HTTP server in a background goroutine
	go func() {
		slog.Info("Server starting", "port", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 5. Block main thread here until an OS signal arrives
	<-ctx.Done()
	slog.Info("Shutdown signal received. Starting graceful shutdown...")

	// 6. Create a 10-second deadline for active connections to drain/finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 7. Stop accepting new requests and wait for existing ones to complete
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exited cleanly")
}
