package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/config"
)

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) VerifySignature(signatureHeader string, rawBody []byte) bool {
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

func (s *Service) ParseEvent(eventType string, rawBody []byte) (*PullRequestEvent, error) {
	if eventType != "pull_request" {
		slog.Info("Ignoring non-PR webhook event", "event_type", eventType)
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}

	var event PullRequestEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		slog.Error("Failed to unmarshal pull_request payload", "error", err)
		return nil, fmt.Errorf("failed to decode JSON payload: %w", err)
	}

	if event.Action != "opened" && event.Action != "synchronize" {
		slog.Info("Ignoring pull request action", "action", event.Action, "pr_number", event.Number)
		return nil, fmt.Errorf("unsupported pull request action: %s", event.Action)
	}

	slog.Info("Successfully parsed pull_request event",
		"action", event.Action,
		"pr_number", event.Number,
		"repo", event.Repository.Name,
		"owner", event.Repository.Owner.Login,
	)

	return &event, nil
}
