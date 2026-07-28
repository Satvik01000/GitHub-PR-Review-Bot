package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	auth       *Authenticator
	httpClient *http.Client
}

type installationTokenResponse struct {
	Token string `json:"token"`
}

func NewClient(auth *Authenticator) *Client {
	return &Client{
		auth: auth,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) GetInstallationToken(ctx context.Context, installationID int64) (string, error) {
	jwtToken, err := c.auth.GenerateJWT()
	if err != nil {
		return "", fmt.Errorf("jwt generation failed: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("installation token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %d getting token: %s", resp.StatusCode, string(body))
	}

	var tokenResp installationTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.Token, nil
}

// GetPullRequestDiff fetches the unified diff string for a PR
func (c *Client) GetPullRequestDiff(ctx context.Context, token, owner, repo string, prNumber int) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", owner, repo, prNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create diff request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("diff request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %d fetching diff: %s", resp.StatusCode, string(body))
	}

	diffBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read diff body: %w", err)
	}

	return string(diffBytes), nil
}
