package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGitHubClient_GetFileSHA(t *testing.T) {
	t.Run("existing file returns sha", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Fatalf("expected GET, got %s", r.Method)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
					t.Fatalf("expected bearer auth header, got %q", got)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"sha":"abc123"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		client := NewGitHubClient(httpClient, "https://api.github.com", "owner", "repo", "main", "test-token")
		sha, exists, err := client.GetFileSHA(context.Background(), "skills/skill/hello/file.txt")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !exists {
			t.Fatal("expected file to exist")
		}
		if sha != "abc123" {
			t.Fatalf("expected sha abc123, got %q", sha)
		}
	})

	t.Run("404 means not found", func(t *testing.T) {
		httpClient := &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
					Header:     make(http.Header),
				}, nil
			}),
		}

		client := NewGitHubClient(httpClient, "https://api.github.com", "owner", "repo", "main", "test-token")
		sha, exists, err := client.GetFileSHA(context.Background(), "skills/skill/hello/file.txt")
		if err != nil {
			t.Fatalf("expected no error on not found, got %v", err)
		}
		if exists {
			t.Fatal("expected file to be not found")
		}
		if sha != "" {
			t.Fatalf("expected empty sha, got %q", sha)
		}
	})
}

func TestGitHubClient_PutFile(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPut {
				t.Fatalf("expected PUT, got %s", r.Method)
			}
			var req struct {
				Message string `json:"message"`
				Content string `json:"content"`
				Branch  string `json:"branch"`
				SHA     string `json:"sha,omitempty"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if req.Message == "" {
				t.Fatal("expected commit message")
			}
			if req.Branch != "main" {
				t.Fatalf("expected branch main, got %q", req.Branch)
			}
			content, err := base64.StdEncoding.DecodeString(req.Content)
			if err != nil {
				t.Fatalf("decode base64 content: %v", err)
			}
			if string(content) != "hello" {
				t.Fatalf("expected uploaded content hello, got %q", string(content))
			}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"content":{"html_url":"https://github.com/owner/repo/blob/main/a.txt"}}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	client := NewGitHubClient(httpClient, "https://api.github.com", "owner", "repo", "main", "test-token")
	url, err := client.PutFile(context.Background(), "a.txt", "add file", []byte("hello"), "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(url, "/blob/main/a.txt") {
		t.Fatalf("expected blob url, got %q", url)
	}
}

func TestGitHubClient_GetFileSHA_RetriesTransientFailures(t *testing.T) {
	var attempts atomic.Int32
	httpClient := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			n := attempts.Add(1)
			if n < 3 {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader("temporary")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"sha":"retry-ok"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	client := NewGitHubClient(httpClient, "https://api.github.com", "owner", "repo", "main", "test-token")
	sha, exists, err := client.GetFileSHA(context.Background(), "skills/skill/retry/file.txt")
	if err != nil {
		t.Fatalf("expected retry success, got error %v", err)
	}
	if !exists {
		t.Fatal("expected file to exist after retry")
	}
	if sha != "retry-ok" {
		t.Fatalf("expected sha retry-ok, got %q", sha)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}
