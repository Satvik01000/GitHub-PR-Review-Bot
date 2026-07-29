package ai

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/config"
)

//go:embed prompts/system_review.md
var systemPrompt string

type Client struct {
	cfg        config.AIConfig
	httpClient *http.Client
}

func NewClient(cfg config.AIConfig) *Client {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
	}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GenerateReview asks the configured AI provider to review a PR diff.
func (c *Client) GenerateReview(ctx context.Context, prTitle, prDescription, diff string) (string, error) {
	if strings.TrimSpace(prDescription) == "" {
		prDescription = "(no description provided)"
	}

	userContent := fmt.Sprintf("PR Title: %s\nPR Description: %s\n\nDiff:\n%s", prTitle, prDescription, diff)

	reqPayload := chatCompletionRequest{
		Model: c.cfg.Model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	}

	jsonBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %w", err)
	}

	endpoint := strings.TrimSuffix(c.cfg.BaseURL, "/") + "/chat/completions"

	maxRetries := 3
	backoff := 2 * time.Second

	for i := 0; i <= maxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBytes))
		if err != nil {
			return "", fmt.Errorf("failed to create http request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if c.cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("ai http request failed: %w", err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var compResp chatCompletionResponse
			if err := json.Unmarshal(body, &compResp); err != nil {
				return "", fmt.Errorf("failed to unmarshal response: %w", err)
			}

			if len(compResp.Choices) > 0 {
				return compResp.Choices[0].Message.Content, nil
			}
			return "No review comments generated.", nil
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			slog.Warn("AI API rate limited or unavailable. Retrying...",
				"attempt", i+1,
				"status", resp.StatusCode,
				"backoff_sec", backoff.Seconds(),
			)

			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
				continue
			}
		}

		return "", fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, string(body))
	}

	return "", fmt.Errorf("exceeded max retries calling AI provider")
}
