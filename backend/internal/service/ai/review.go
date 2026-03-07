package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// ReviewResult contains the structured output from AI skill review.
type ReviewResult struct {
	Approved      bool   `json:"approved"`
	Feedback      string `json:"feedback"`
	AIDescription string `json:"ai_description"`
	FuncSummary   string `json:"func_summary"`
}

// ReviewSkill sends the skill to AI for quality/safety review and returns a structured result.
func (s *Service) ReviewSkill(name, resourceType, description, content string) (ReviewResult, error) {
	if s.cfg.OpenAIKey == "" {
		return ReviewResult{
			Approved:      true,
			Feedback:      "AI review not configured (missing OPENAI_API_KEY), auto-approved.",
			AIDescription: "Security: Not scanned.\nFunction: " + name,
			FuncSummary:   name,
		}, nil
	}

	prompt := fmt.Sprintf(ReviewUserPromptTemplate,
		name,
		resourceType,
		description,
		truncate(content, 2000),
	)

	messages := []ChatMessage{
		{Role: "system", Content: ReviewSystemPrompt},
		{Role: "user", Content: prompt},
	}

	responseText, err := s.callChat(context.Background(), messages)
	if err != nil {
		return ReviewResult{}, err
	}

	responseText = cleanJSON(responseText)

	var result ReviewResult
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return ReviewResult{}, fmt.Errorf("parse AI review JSON failed: %w", err)
	}

	return result, nil
}
