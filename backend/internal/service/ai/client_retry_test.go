package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"skill-hub/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCallStream_RetryOnInitialEOF(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	streamBody := "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: [DONE]\n\n"

	svc := NewService(&config.Config{
		OpenAIKey:     "test-key",
		OpenAIBaseURL: "https://api.openai.com/v1",
		OpenAIModel:   "test-model",
	})
	svc.httpClient.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		if attempts.Add(1) == 1 {
			return nil, io.EOF
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(streamBody)),
			Header:     make(http.Header),
		}, nil
	})

	var got strings.Builder
	err := svc.callStream(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, func(chunk string) error {
		got.WriteString(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("expected retry success, got error: %v", err)
	}
	if got.String() != "hello" {
		t.Fatalf("expected streamed content hello, got %q", got.String())
	}
	if attempts.Load() != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts.Load())
	}
}
