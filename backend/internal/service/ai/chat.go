package ai

import (
	"context"
	"fmt"
)

// ChatRecommendStream sends a user message to AI with skill catalog context and streams the response.
func (s *Service) ChatRecommendStream(ctx context.Context, userMessage string, skillsJSON string, writer func(string) error) error {
	if s.cfg.OpenAIKey == "" {
		return writer("AI recommendation is not configured (missing OPENAI_API_KEY). Please set the environment variable to enable AI features.")
	}

	systemPrompt := fmt.Sprintf(ChatSystemPromptTemplate, skillsJSON)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	return s.callStream(ctx, messages, writer)
}
