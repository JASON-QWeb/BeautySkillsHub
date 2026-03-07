package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"skill-hub/internal/config"
)

const defaultAIHTTPTimeout = 120 * time.Second

// Service handles all AI-related operations via OpenAI-compatible API.
type Service struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: defaultAIHTTPTimeout,
		},
	}
}

// ChatMessage represents a single message in an OpenAI conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request body for OpenAI chat completions.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

// ChatResponse is the non-streaming response from OpenAI.
type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

// StreamChunk represents a single SSE chunk from OpenAI streaming.
type StreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// callChat sends a non-streaming chat completion request and returns the response text.
func (s *Service) callChat(ctx context.Context, messages []ChatMessage) (string, error) {
	req := ChatRequest{
		Model:    s.cfg.OpenAIModel,
		Messages: messages,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, s.httpClient.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.cfg.OpenAIBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.OpenAIKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// callStream sends a streaming chat completion request.
func (s *Service) callStream(ctx context.Context, messages []ChatMessage, writer func(string) error) error {
	req := ChatRequest{
		Model:    s.cfg.OpenAIModel,
		Messages: messages,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, s.httpClient.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.cfg.OpenAIBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.OpenAIKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	buf := make([]byte, 4096)
	var leftover string

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			data := leftover + string(buf[:n])
			leftover = ""

			lines := strings.Split(data, "\n")
			for i, line := range lines {
				if i == len(lines)-1 && !strings.HasSuffix(data, "\n") {
					leftover = line
					continue
				}

				line = strings.TrimSpace(line)
				if line == "" || line == "data: [DONE]" {
					continue
				}

				if strings.HasPrefix(line, "data: ") {
					jsonData := strings.TrimPrefix(line, "data: ")
					var chunk StreamChunk
					if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
						continue
					}
					if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
						if err := writer(chunk.Choices[0].Delta.Content); err != nil {
							return err
						}
					}
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	return nil
}

// cleanJSON strips markdown code fences from AI response.
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
