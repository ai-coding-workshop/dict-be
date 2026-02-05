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

type OpenAIConfig struct {
	BaseURL    string
	Token      string
	Model      string
	HTTPClient *http.Client
}

type OpenAIClient struct {
	baseURL    string
	token      string
	model      string
	httpClient *http.Client
}

func NewOpenAIClient(cfg OpenAIConfig) (*OpenAIClient, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, errors.New("openai base url is required")
	}
	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, errors.New("openai token is required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, errors.New("openai model is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	return &OpenAIClient{
		baseURL:    baseURL,
		token:      token,
		model:      model,
		httpClient: client,
	}, nil
}

func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	payload := openAIChatRequest{
		Model:    c.resolveModel(req.Model),
		Messages: req.Messages,
	}
	var resp openAIChatResponse
	if err := c.do(ctx, payload, &resp); err != nil {
		return ChatResponse{}, err
	}
	if len(resp.Choices) == 0 {
		return ChatResponse{}, errors.New("openai response has no choices")
	}
	return ChatResponse{
		Content:      resp.Choices[0].Message.Content,
		Model:        resp.Model,
		FinishReason: resp.Choices[0].FinishReason,
	}, nil
}

func (c *OpenAIClient) ChatStream(ctx context.Context, req ChatRequest, handle StreamHandler) (ChatResponse, error) {
	payload := openAIChatRequest{
		Model:    c.resolveModel(req.Model),
		Messages: req.Messages,
		Stream:   true,
	}
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}
	endpoint := buildChatEndpoint(c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Accept", "text/event-stream")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("openai request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return ChatResponse{}, readOpenAIError(httpResp.Body, httpResp.StatusCode)
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
		var chunk openAIChatResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return ChatResponse{}, fmt.Errorf("decode stream chunk: %w", err)
		}
		if chunk.Error != nil {
			return ChatResponse{}, fmt.Errorf("openai error: %s", chunk.Error.Message)
		}
		if chunk.Model != "" {
			model = chunk.Model
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		if chunk.Choices[0].FinishReason != "" {
			finishReason = chunk.Choices[0].FinishReason
		}
		delta := chunk.Choices[0].Delta.Content
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

func (c *OpenAIClient) resolveModel(override string) string {
	if strings.TrimSpace(override) == "" {
		return c.model
	}
	return override
}

func (c *OpenAIClient) do(ctx context.Context, payload openAIChatRequest, out *openAIChatResponse) error {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	endpoint := buildChatEndpoint(c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("openai request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return readOpenAIError(httpResp.Body, httpResp.StatusCode)
	}
	if err := json.NewDecoder(httpResp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if out.Error != nil {
		return fmt.Errorf("openai error: %s", out.Error.Message)
	}
	return nil
}

func buildChatEndpoint(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

func readOpenAIError(body io.Reader, status int) error {
	var resp openAIChatResponse
	_ = json.NewDecoder(body).Decode(&resp)
	if resp.Error != nil && resp.Error.Message != "" {
		return fmt.Errorf("openai request failed: %s (status %d)", resp.Error.Message, status)
	}
	return fmt.Errorf("openai request failed with status %d", status)
}

type openAIChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message      Message `json:"message"`
		Delta        Message `json:"delta"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}
