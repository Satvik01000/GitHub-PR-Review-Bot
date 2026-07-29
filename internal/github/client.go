package github

import (
	"bytes"
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

type prReviewRequest struct {
	Body  string `json:"body"`
	Event string `json:"event"`
}

type checkRunCreateRequest struct {
	Name    string `json:"name"`
	HeadSHA string `json:"head_sha"`
	Status  string `json:"status"`
}

type checkRunUpdateRequest struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type checkRunResponse struct {
	ID int64 `json:"id"`
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

func (c *Client) PostPRReview(ctx context.Context, token, owner, repo string, prNumber int, commentBody string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/reviews", owner, repo, prNumber)

	reqPayload := prReviewRequest{
		Body:  commentBody,
		Event: "COMMENT",
	}

	jsonBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal review request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create post review request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post review request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d posting review: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateCheckRun triggers the initial spinner UI on the PR checks section
func (c *Client) CreateCheckRun(ctx context.Context, token, owner, repo, headSHA string) (int64, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/check-runs", owner, repo)

	payload := checkRunCreateRequest{
		Name:    "Code-Bot AI Review",
		HeadSHA: headSHA,
		Status:  "in_progress",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed creating check run status %d: %s", resp.StatusCode, string(body))
	}

	var checkResp checkRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&checkResp); err != nil {
		return 0, err
	}

	return checkResp.ID, nil
}

// CompleteCheckRun resolves the spinner UI (always success/neutral so it doesn't block merge)
func (c *Client) CompleteCheckRun(ctx context.Context, token, owner, repo string, checkRunID int64, conclusion string) error {
	if checkRunID == 0 {
		return nil
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/check-runs/%d", owner, repo, checkRunID)

	payload := checkRunUpdateRequest{
		Status:     "completed",
		Conclusion: conclusion,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
