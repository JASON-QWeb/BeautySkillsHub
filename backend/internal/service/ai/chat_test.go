package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"skill-hub/internal/config"
)

func TestChatRecommendStream_UsesRequestContext(t *testing.T) {
	t.Parallel()

	var hitServer atomic.Bool
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hitServer.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"delta":{"content":"hello"}}]}`))
	}))
	defer mock.Close()

	svc := NewService(&config.Config{
		OpenAIKey:     "test-key",
		OpenAIBaseURL: mock.URL,
		OpenAIModel:   "test-model",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.ChatRecommendStream(ctx, "hi", "[]", func(string) error { return nil })
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
	if hitServer.Load() {
		t.Fatal("expected canceled context to stop request before it hits server")
	}
}
