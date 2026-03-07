package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	githubRequestMaxAttempts = 3
	githubRetryBaseDelay     = 200 * time.Millisecond
	githubRetryMaxDelay      = 2 * time.Second
)

// GitHubContentClient wraps minimal GitHub contents APIs needed for sync.
type GitHubContentClient interface {
	GetFileSHA(ctx context.Context, path string) (sha string, exists bool, err error)
	PutFile(ctx context.Context, path, message string, content []byte, sha string) (htmlURL string, err error)
	DeleteFile(ctx context.Context, path, message, sha string) error
	ListDir(ctx context.Context, path string) ([]string, error)
}

type GitHubClient struct {
	httpClient *http.Client
	baseURL    string
	owner      string
	repo       string
	branch     string
	token      string
}

func NewGitHubClient(httpClient *http.Client, baseURL, owner, repo, branch, token string) *GitHubClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "https://api.github.com"
	}
	if strings.TrimSpace(branch) == "" {
		branch = "main"
	}

	return &GitHubClient{
		httpClient: httpClient,
		baseURL:    base,
		owner:      owner,
		repo:       repo,
		branch:     branch,
		token:      token,
	}
}

func (c *GitHubClient) GetFileSHA(ctx context.Context, filePath string) (string, bool, error) {
	endpoint := c.contentsEndpoint(filePath)
	resp, err := c.doWithRetry(ctx, http.MethodGet, endpoint, nil, func(req *http.Request) {
		req.URL.RawQuery = "ref=" + url.QueryEscape(c.branch)
	})
	if err != nil {
		return "", false, fmt.Errorf("github GET request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("github GET %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", false, fmt.Errorf("decode github GET response: %w", err)
	}
	if payload.SHA == "" {
		return "", false, fmt.Errorf("github GET response missing sha")
	}
	return payload.SHA, true, nil
}

func (c *GitHubClient) PutFile(ctx context.Context, filePath, message string, content []byte, sha string) (string, error) {
	endpoint := c.contentsEndpoint(filePath)
	payload := struct {
		Message string `json:"message"`
		Content string `json:"content"`
		Branch  string `json:"branch"`
		SHA     string `json:"sha,omitempty"`
	}{
		Message: message,
		Content: base64.StdEncoding.EncodeToString(content),
		Branch:  c.branch,
		SHA:     strings.TrimSpace(sha),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal github PUT payload: %w", err)
	}

	resp, err := c.doWithRetry(ctx, http.MethodPut, endpoint, body, nil)
	if err != nil {
		return "", fmt.Errorf("github PUT request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github PUT %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var respPayload struct {
		Content struct {
			HTMLURL string `json:"html_url"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respPayload); err != nil {
		return "", fmt.Errorf("decode github PUT response: %w", err)
	}
	if respPayload.Content.HTMLURL != "" {
		return respPayload.Content.HTMLURL, nil
	}

	return fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", c.owner, c.repo, c.branch, strings.Trim(filePath, "/")), nil
}

func (c *GitHubClient) DeleteFile(ctx context.Context, filePath, message, sha string) error {
	endpoint := c.contentsEndpoint(filePath)
	payload := struct {
		Message string `json:"message"`
		SHA     string `json:"sha"`
		Branch  string `json:"branch"`
	}{
		Message: message,
		SHA:     sha,
		Branch:  c.branch,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal github DELETE payload: %w", err)
	}

	resp, err := c.doWithRetry(ctx, http.MethodDelete, endpoint, body, nil)
	if err != nil {
		return fmt.Errorf("github DELETE request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github DELETE %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func (c *GitHubClient) ListDir(ctx context.Context, dirPath string) ([]string, error) {
	endpoint := c.contentsEndpoint(dirPath)
	resp, err := c.doWithRetry(ctx, http.MethodGet, endpoint, nil, func(req *http.Request) {
		req.URL.RawQuery = "ref=" + url.QueryEscape(c.branch)
	})
	if err != nil {
		return nil, fmt.Errorf("github GET request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github GET %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, _ := io.ReadAll(resp.Body)

	// GitHub returns an array for directories, an object for files
	var items []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &items); err != nil {
		// Might be a single file object, not a directory
		var single struct {
			Path string `json:"path"`
			Type string `json:"type"`
		}
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return nil, fmt.Errorf("decode github list response: %w", err)
		}
		if single.Path != "" {
			return []string{single.Path}, nil
		}
		return nil, nil
	}

	var paths []string
	for _, item := range items {
		paths = append(paths, item.Path)
	}
	return paths, nil
}

func (c *GitHubClient) applyHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func (c *GitHubClient) contentsEndpoint(filePath string) string {
	return fmt.Sprintf(
		"%s/repos/%s/%s/contents/%s",
		c.baseURL,
		url.PathEscape(c.owner),
		url.PathEscape(c.repo),
		escapeGitHubPath(filePath),
	)
}

func escapeGitHubPath(filePath string) string {
	parts := strings.Split(strings.Trim(filePath, "/"), "/")
	encoded := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		encoded = append(encoded, url.PathEscape(part))
	}
	return strings.Join(encoded, "/")
}

func (c *GitHubClient) doWithRetry(
	ctx context.Context,
	method string,
	endpoint string,
	body []byte,
	configure func(*http.Request),
) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var lastResp *http.Response
	var lastErr error

	for attempt := 1; attempt <= githubRequestMaxAttempts; attempt++ {
		var requestBody io.Reader
		if body != nil {
			requestBody = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, requestBody)
		if err != nil {
			return nil, fmt.Errorf("create github %s request: %w", method, err)
		}
		c.applyHeaders(req)
		if configure != nil {
			configure(req)
		}

		resp, err := c.httpClient.Do(req)
		lastResp = resp
		lastErr = err

		if !shouldRetryGitHubRequest(resp, err) || attempt == githubRequestMaxAttempts {
			return resp, err
		}

		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		delay := githubRetryDelay(attempt, resp)
		if waitErr := waitWithContext(ctx, delay); waitErr != nil {
			return nil, waitErr
		}
	}

	return lastResp, lastErr
}

func shouldRetryGitHubRequest(resp *http.Response, err error) bool {
	if err != nil {
		return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
	}
	if resp == nil {
		return false
	}
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
}

func githubRetryDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if retryAfter := strings.TrimSpace(resp.Header.Get("Retry-After")); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}

	if attempt <= 0 {
		attempt = 1
	}
	delay := githubRetryBaseDelay * time.Duration(1<<(attempt-1))
	if delay > githubRetryMaxDelay {
		return githubRetryMaxDelay
	}
	return delay
}

func waitWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
