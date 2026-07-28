package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/config"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/domain"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/worker"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrIgnoredEvent     = errors.New("event ignored")
	ErrQueueFull        = errors.New("worker queue full")
)

type Service struct {
	cfg        *config.Config
	workerPool *worker.Pool
}

func NewService(cfg *config.Config, workerPool *worker.Pool) *Service {
	return &Service{
		cfg:        cfg,
		workerPool: workerPool,
	}
}

func (s *Service) ProcessWebhook(signatureHeader, eventType string, rawBody []byte) error {
	if !s.verifySignature(signatureHeader, rawBody) {
		return ErrInvalidSignature
	}

	event, err := s.parseEvent(eventType, rawBody)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrIgnoredEvent, err)
	}

	enqueued := s.workerPool.Enqueue(worker.Job{Event: event})
	if !enqueued {
		return ErrQueueFull
	}

	return nil
}

func (s *Service) verifySignature(signatureHeader string, rawBody []byte) bool {
	if signatureHeader == "" || !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}
	signatureHex := strings.TrimPrefix(signatureHeader, "sha256=")
	gotMac, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}

	expectedMac := hmac.New(sha256.New, []byte(s.cfg.Server.WebhookSecret))
	expectedMac.Write(rawBody)
	return hmac.Equal(gotMac, expectedMac.Sum(nil))
}

func (s *Service) parseEvent(eventType string, rawBody []byte) (*domain.PullRequestEvent, error) {
	if eventType != "pull_request" {
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}

	var event domain.PullRequestEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return nil, fmt.Errorf("failed to decode JSON payload: %w", err)
	}

	if event.Action != "opened" && event.Action != "synchronize" {
		return nil, fmt.Errorf("unsupported pull request action: %s", event.Action)
	}

	return &event, nil
}
