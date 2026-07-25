package server

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/config"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/webhook"
)

type Server struct {
	cfg            *config.Config
	webhookService *webhook.Service
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		cfg:            cfg,
		webhookService: webhook.NewService(cfg), // Fixed: Initialize webhookService here
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

	signatureHeader := r.Header.Get("X-Hub-Signature-256")

	if !s.webhookService.VerifySignature(signatureHeader, rawBody) {
		slog.Error("Failed to verify webhook signature", "signature", signatureHeader)
		http.Error(w, "Unauthorized: Invalid Signature", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	event, err := s.webhookService.ParseEvent(eventType, rawBody)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Event ignored"))
		return
	}

	slog.Info("Processing PR review request", "pr_number", event.Number, "repo", event.Repository.Name)

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Webhook received and queued")); err != nil {
		slog.Error("Failed to write response body", "error", err)
	}
}
