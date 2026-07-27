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
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/worker"
)

func main() {
	// Initialize default structured logger outputting text to stdout
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Create a context that listens for OS interrupt signals (Ctrl+C, SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Initialize and start the Worker Pool using config values
	workerPool := worker.NewPool(cfg.Worker.MaxWorkers, cfg.Worker.QueueSize)
	workerPool.Start(ctx)

	// 4. Initialize router & server with workerPool dependency injection
	srv := server.NewServer(cfg, workerPool)
	httpServer := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: srv.RegisterRoutes(),
	}

	// 5. Start HTTP server in a background goroutine
	go func() {
		slog.Info("Server starting", "port", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 6. Block main thread here until an OS signal arrives
	<-ctx.Done()
	slog.Info("Shutdown signal received. Starting graceful shutdown...")

	// 7. Stop accepting new jobs and wait for workers to finish active jobs
	workerPool.Stop()

	// 8. Create a 10-second deadline for active HTTP connections to drain
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 9. Shutdown HTTP server cleanly
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exited cleanly")
}
