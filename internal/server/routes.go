package server

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/config"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/webhook"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/worker"
)

type Server struct {
	cfg            *config.Config
	webhookService *webhook.Service
}

func NewServer(cfg *config.Config, workerPool *worker.Pool) *Server {
	return &Server{
		cfg:            cfg,
		webhookService: webhook.NewService(cfg, workerPool),
	}
}

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /webhook", s.handleWebhook)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read request body", "error", err)
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	signature := r.Header.Get("X-Hub-Signature-256")
	eventType := r.Header.Get("X-GitHub-Event")

	err = s.webhookService.ProcessWebhook(signature, eventType, rawBody)
	if err != nil {
		switch {
		case errors.Is(err, webhook.ErrInvalidSignature):
			slog.Error("Webhook signature validation failed")
			http.Error(w, "Unauthorized: Invalid Signature", http.StatusUnauthorized)
		case errors.Is(err, webhook.ErrIgnoredEvent):
			slog.Info("Webhook event ignored", "reason", err)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Event ignored"))
		case errors.Is(err, webhook.ErrQueueFull):
			slog.Error("Worker queue full, rejecting request")
			http.Error(w, "Server Busy: Worker queue full", http.StatusServiceUnavailable)
		default:
			slog.Error("Unexpected webhook processing error", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and queued"))
}
