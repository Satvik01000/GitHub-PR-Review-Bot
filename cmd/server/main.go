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

	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/ai"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/config"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/github"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/server"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/worker"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Initialize GitHub Authenticator and Client
	auth, err := github.NewAuthenticator(cfg.GitHub.AppID, cfg.GitHub.PrivateKeyPath)
	if err != nil {
		slog.Error("Failed to initialize GitHub authenticator", "error", err)
		os.Exit(1)
	}
	githubClient := github.NewClient(auth)

	// 3. Initialize AI Client
	aiClient := ai.NewClient(cfg.AI)

	// 4. Create context listening for OS signals (SIGINT, SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 5. Initialize and start worker pool
	workerPool := worker.NewPool(cfg.Worker.MaxWorkers, cfg.Worker.QueueSize, githubClient, aiClient)
	workerPool.Start(ctx)

	// 6. Initialize server and routes
	srv := server.NewServer(cfg, workerPool)
	httpServer := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: srv.RegisterRoutes(),
	}

	// 7. Start HTTP server
	go func() {
		slog.Info("Server starting", "port", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 8. Wait for shutdown signal
	<-ctx.Done()
	slog.Info("Shutdown signal received. Starting graceful shutdown...")

	workerPool.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exited cleanly")
}
