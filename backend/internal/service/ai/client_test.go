package ai

import (
	"testing"

	"skill-hub/internal/config"
)

func TestNewService_UsesDefaultHTTPTimeout(t *testing.T) {
	svc := NewService(&config.Config{})

	if svc.httpClient == nil {
		t.Fatal("expected http client to be initialized")
	}
	if svc.httpClient.Timeout != defaultAIHTTPTimeout {
		t.Fatalf("expected default timeout %v, got %v", defaultAIHTTPTimeout, svc.httpClient.Timeout)
	}
}
