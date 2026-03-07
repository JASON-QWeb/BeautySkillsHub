package ai

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"skill-hub/internal/config"
)

func TestReviewSkill_ReturnsErrorWhenAIResponseIsInvalidJSON(t *testing.T) {
	t.Parallel()

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"this is not json"}}]}`))
	}))
	defer mock.Close()

	svc := NewService(&config.Config{
		OpenAIKey:     "test-key",
		OpenAIBaseURL: mock.URL,
		OpenAIModel:   "test-model",
	})

	_, err := svc.ReviewSkill("demo", "skill", "desc", "content")
	if err == nil {
		t.Fatal("expected ReviewSkill to fail when AI returns invalid JSON")
	}
}
