package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"skill-hub/internal/config"
)

type AIService struct {
	cfg *config.Config
}

func NewAIService(cfg *config.Config) *AIService {
	return &AIService{cfg: cfg}
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

// ReviewSkill sends the skill info to OpenAI for review and returns (approved, feedback, error).
func (s *AIService) ReviewSkill(name, description, content string) (bool, string, error) {
	if s.cfg.OpenAIKey == "" {
		// If no API key configured, auto-approve with info message
		return true, "AI 审核未配置（缺少 OPENAI_API_KEY），自动通过。", nil
	}

	prompt := fmt.Sprintf(`你是一个 Skill 审核员。请审核以下 Skill 的质量和安全性。

Skill 名称: %s
Skill 描述: %s
Skill 内容（前2000字符）:
%s

请评估此 Skill 是否：
1. 内容描述清晰有意义
2. 没有恶意代码或不安全内容
3. 名称和描述与内容相符

请用以下 JSON 格式回复（不要有额外文字）：
{"approved": true/false, "feedback": "你的审核意见"}`, name, description, truncate(content, 2000))

	req := ChatRequest{
		Model: s.cfg.OpenAIModel,
		Messages: []ChatMessage{
			{Role: "system", Content: "你是一个专业的技能审核助手，只输出JSON格式的审核结果。"},
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return false, "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.cfg.OpenAIBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return false, "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.OpenAIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return false, "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return false, "", fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return false, "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return false, "", fmt.Errorf("no choices in response")
	}

	// Parse the JSON result
	responseText := chatResp.Choices[0].Message.Content
	responseText = strings.TrimSpace(responseText)
	// Remove potential markdown code fences
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	var result struct {
		Approved bool   `json:"approved"`
		Feedback string `json:"feedback"`
	}
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		// If JSON parsing fails, approve with raw feedback
		return true, responseText, nil
	}

	return result.Approved, result.Feedback, nil
}

// ChatRecommendStream sends a chat message to OpenAI with skill context and streams the response.
// The writer func is called for each text chunk received.
func (s *AIService) ChatRecommendStream(userMessage string, skillsJSON string, writer func(string) error) error {
	if s.cfg.OpenAIKey == "" {
		return writer("AI 推荐功能未配置（缺少 OPENAI_API_KEY）。请设置环境变量 OPENAI_API_KEY 以启用 AI 功能。")
	}

	systemPrompt := fmt.Sprintf(`你是 Skill Hub 的 AI 助手。你可以根据用户的需求推荐合适的 Skill。
以下是平台上当前可用的 Skill 列表（JSON 格式）：

%s

请根据用户的描述，推荐最相关的 Skill，并解释推荐理由。如果没有匹配的 Skill，告诉用户并建议他们上传新的 Skill。
回复请用中文，保持友好和专业。`, skillsJSON)

	req := ChatRequest{
		Model: s.cfg.OpenAIModel,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Stream: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.cfg.OpenAIBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.OpenAIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Read SSE stream
	reader := io.Reader(resp.Body)
	buf := make([]byte, 4096)
	var leftover string

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			data := leftover + string(buf[:n])
			leftover = ""

			lines := strings.Split(data, "\n")
			for i, line := range lines {
				// If last line might be incomplete
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
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
