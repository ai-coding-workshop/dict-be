package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultAnthropicVersion   = "2023-06-01"
	defaultAnthropicMaxTokens = 1024
)

type AnthropicConfig struct {
	BaseURL    string
	Token      string
	Model      string
	Version    string
	MaxTokens  int
	HTTPClient *http.Client
}

type AnthropicClient struct {
	baseURL    string
	token      string
	model      string
	version    string
	maxTokens  int
	httpClient *http.Client
}

func NewAnthropicClient(cfg AnthropicConfig) (*AnthropicClient, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, errors.New("anthropic base url is required")
	}
	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, errors.New("anthropic token is required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, errors.New("anthropic model is required")
	}
	version := strings.TrimSpace(cfg.Version)
	if version == "" {
		version = defaultAnthropicVersion
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultAnthropicMaxTokens
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	return &AnthropicClient{
		baseURL:    baseURL,
		token:      token,
		model:      model,
		version:    version,
		maxTokens:  maxTokens,
		httpClient: client,
	}, nil
}

func (c *AnthropicClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	messages, system := splitAnthropicMessages(req.Messages)
	payload := anthropicChatRequest{
		Model:     c.resolveModel(req.Model),
		Messages:  messages,
		System:    system,
		MaxTokens: c.maxTokens,
	}
	var resp anthropicChatResponse
	if err := c.do(ctx, payload, &resp); err != nil {
		return ChatResponse{}, err
	}
	content := flattenAnthropicContent(resp.Content)
	return ChatResponse{
		Content:      content,
		Model:        resp.Model,
		FinishReason: resp.StopReason,
	}, nil
}

func (c *AnthropicClient) ChatStream(ctx context.Context, req ChatRequest, handle StreamHandler) (ChatResponse, error) {
	messages, system := splitAnthropicMessages(req.Messages)
	payload := anthropicChatRequest{
		Model:     c.resolveModel(req.Model),
		Messages:  messages,
		System:    system,
		MaxTokens: c.maxTokens,
		Stream:    true,
	}
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}
	endpoint := buildAnthropicEndpoint(c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-api-key", c.token)
	httpReq.Header.Set("anthropic-version", c.version)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("anthropic request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return ChatResponse{}, readAnthropicError(httpResp.Body, httpResp.StatusCode)
	}

	var content strings.Builder
	var finishReason string
	var model string

	scanner := bufio.NewScanner(httpResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return ChatResponse{}, fmt.Errorf("decode stream chunk: %w", err)
		}
		if event.Type == "error" && event.Error != nil {
			return ChatResponse{}, fmt.Errorf("anthropic error: %s", event.Error.Message)
		}
		if event.Type == "message_start" && event.Message != nil && event.Message.Model != "" {
			model = event.Message.Model
		}
		if event.Type == "message_delta" && event.StopReason != "" {
			finishReason = event.StopReason
		}
		if event.Type != "content_block_delta" || event.Delta == nil {
			continue
		}
		delta := event.Delta.Text
		if delta == "" {
			continue
		}
		content.WriteString(delta)
		if handle != nil {
			if err := handle(delta); err != nil {
				return ChatResponse{}, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ChatResponse{}, fmt.Errorf("read stream: %w", err)
	}
	return ChatResponse{
		Content:      content.String(),
		Model:        model,
		FinishReason: finishReason,
	}, nil
}

func (c *AnthropicClient) resolveModel(override string) string {
	if strings.TrimSpace(override) == "" {
		return c.model
	}
	return override
}

func (c *AnthropicClient) do(ctx context.Context, payload anthropicChatRequest, out *anthropicChatResponse) error {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	endpoint := buildAnthropicEndpoint(c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.token)
	httpReq.Header.Set("anthropic-version", c.version)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("anthropic request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return readAnthropicError(httpResp.Body, httpResp.StatusCode)
	}
	if err := json.NewDecoder(httpResp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if out.Error != nil {
		return fmt.Errorf("anthropic error: %s", out.Error.Message)
	}
	return nil
}

func buildAnthropicEndpoint(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/messages"
	}
	return base + "/v1/messages"
}

func readAnthropicError(body io.Reader, status int) error {
	var resp anthropicChatResponse
	_ = json.NewDecoder(body).Decode(&resp)
	if resp.Error != nil && resp.Error.Message != "" {
		return fmt.Errorf("anthropic request failed: %s (status %d)", resp.Error.Message, status)
	}
	return fmt.Errorf("anthropic request failed with status %d", status)
}

func splitAnthropicMessages(messages []Message) ([]Message, string) {
	if len(messages) == 0 {
		return messages, ""
	}
	first := messages[0]
	if first.Role != "system" {
		return messages, ""
	}
	return messages[1:], first.Content
}

func flattenAnthropicContent(blocks []anthropicContent) string {
	if len(blocks) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, block := range blocks {
		if block.Type != "text" {
			continue
		}
		builder.WriteString(block.Text)
	}
	return builder.String()
}

type anthropicChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	System    string    `json:"system,omitempty"`
	MaxTokens int       `json:"max_tokens"`
	Stream    bool      `json:"stream,omitempty"`
}

type anthropicChatResponse struct {
	ID         string             `json:"id"`
	Model      string             `json:"model"`
	Content    []anthropicContent `json:"content"`
	StopReason string             `json:"stop_reason"`
	Error      *anthropicError    `json:"error,omitempty"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type anthropicStreamEvent struct {
	Type       string          `json:"type"`
	Message    *anthropicEvent `json:"message,omitempty"`
	Delta      *anthropicDelta `json:"delta,omitempty"`
	StopReason string          `json:"stop_reason,omitempty"`
	Error      *anthropicError `json:"error,omitempty"`
}

type anthropicEvent struct {
	Model string `json:"model"`
}

type anthropicDelta struct {
	Text string `json:"text"`
}
