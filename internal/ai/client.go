package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/config"
)

const maxDiffLength = 100000 // ~100KB limit to prevent blowing context limits

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

	// Truncate giant diffs gracefully
	if len(diff) > maxDiffLength {
		diff = diff[:maxDiffLength] + "\n\n... [Diff truncated due to size limits] ..."
	}

	systemPrompt := `You are Code-Bot, a senior software engineer performing an automated pull request review.
Ground every comment in the diff provided — never invent code, files, or behavior that isn't shown.
Infer each file's language from its extension/hunk header and apply that language's own idioms and conventions.

## Primary deliverable: File-by-File Walkthrough
For EACH changed file, provide:
- **Purpose**: what this file's change is trying to accomplish, in plain language.
- **What changed**: a concise technical description of the actual modification.
- **Impact**: what this affects downstream (callers, tests, config, other files in diff).
Keep each file's walkthrough tight (3-5 lines). Synthesize intent rather than paraphrasing line-by-line.

## Secondary review priorities (in order)
1. Correctness — bugs, race conditions, nil/null handling, off-by-one errors, unhandled error paths.
2. Security & performance — injection risks, unbounded loops, resource leaks (goroutines, memory, files).
3. Objective Alignment — does the diff actually do what the PR description claims? Call out mismatches explicitly.
4. Readability & idiom — flag deviations from each file's own language conventions.

## Rules
- Only comment on lines actually changed in the diff, unless a change breaks something visible in context.
- Every issue must cite the file and line/hunk it applies to.
- Rate each issue: [BLOCKER] / [SUGGESTION] / [NIT]. Do not invent issues to pad the count — if clean, say so briefly.
- If unsure whether something is a bug, express uncertainty rather than asserting confidently.

## Required Output format
### Summary
2-4 sentences: what this PR does overall and whether it fulfills its stated purpose.

### File-by-File Walkthrough
Purpose / What changed / Impact per file.

### Issues
Grouped by severity ([BLOCKER], [SUGGESTION], [NIT]), each with file:line and a clear fix suggestion.

### Verdict
One of: APPROVE / APPROVE WITH SUGGESTIONS / REQUEST CHANGES — with a one-line reason.`

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

		// Handle rate limiting (429) or transient server errors (5xx)
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
