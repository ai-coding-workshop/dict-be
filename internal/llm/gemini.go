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
	"net/url"
	"path"
	"strings"
)

type GeminiConfig struct {
	BaseURL    string
	Token      string
	Model      string
	HTTPClient *http.Client
}

type GeminiClient struct {
	baseURL    string
	token      string
	model      string
	httpClient *http.Client
}

func NewGeminiClient(cfg GeminiConfig) (*GeminiClient, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, errors.New("gemini base url is required")
	}
	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, errors.New("gemini token is required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, errors.New("gemini model is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	return &GeminiClient{
		baseURL:    baseURL,
		token:      token,
		model:      model,
		httpClient: client,
	}, nil
}

func (c *GeminiClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	contents, system := buildGeminiContents(req.Messages)
	payload := geminiGenerateContentRequest{
		Contents:          contents,
		SystemInstruction: system,
	}
	var resp geminiGenerateContentResponse
	if err := c.do(ctx, payload, c.resolveModel(req.Model), false, &resp); err != nil {
		return ChatResponse{}, err
	}
	if len(resp.Candidates) == 0 {
		return ChatResponse{}, errors.New("gemini response has no candidates")
	}
	content := flattenGeminiContent(resp.Candidates[0].Content)
	return ChatResponse{
		Content:      content,
		Model:        resp.ModelVersion,
		FinishReason: resp.Candidates[0].FinishReason,
	}, nil
}

func (c *GeminiClient) ChatStream(ctx context.Context, req ChatRequest, handle StreamHandler) (ChatResponse, error) {
	contents, system := buildGeminiContents(req.Messages)
	payload := geminiGenerateContentRequest{
		Contents:          contents,
		SystemInstruction: system,
	}
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}
	model := c.resolveModel(req.Model)
	endpoint, err := buildGeminiEndpoint(c.baseURL, model, true, c.token)
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("gemini request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return ChatResponse{}, readGeminiError(httpResp.Body, httpResp.StatusCode)
	}

	var content strings.Builder
	var finishReason string
	var modelVersion string

	scanner := bufio.NewScanner(httpResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		data := line
		if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if data == "[DONE]" {
			break
		}
		if !strings.HasPrefix(data, "{") {
			continue
		}
		var chunk geminiGenerateContentResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return ChatResponse{}, fmt.Errorf("decode stream chunk: %w", err)
		}
		if chunk.Error != nil {
			return ChatResponse{}, fmt.Errorf("gemini error: %s", chunk.Error.Message)
		}
		if chunk.ModelVersion != "" {
			modelVersion = chunk.ModelVersion
		}
		if len(chunk.Candidates) == 0 {
			continue
		}
		if chunk.Candidates[0].FinishReason != "" {
			finishReason = chunk.Candidates[0].FinishReason
		}
		delta := flattenGeminiContent(chunk.Candidates[0].Content)
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
		Model:        modelVersion,
		FinishReason: finishReason,
	}, nil
}

func (c *GeminiClient) resolveModel(override string) string {
	if strings.TrimSpace(override) == "" {
		return c.model
	}
	return override
}

func (c *GeminiClient) do(ctx context.Context, payload geminiGenerateContentRequest, model string, stream bool, out *geminiGenerateContentResponse) error {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	endpoint, err := buildGeminiEndpoint(c.baseURL, model, stream, c.token)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("gemini request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return readGeminiError(httpResp.Body, httpResp.StatusCode)
	}
	if err := json.NewDecoder(httpResp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if out.Error != nil {
		return fmt.Errorf("gemini error: %s", out.Error.Message)
	}
	return nil
}

func buildGeminiEndpoint(baseURL, model string, stream bool, token string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", errors.New("gemini base url is required")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return "", errors.New("gemini model is required")
	}
	if strings.TrimSpace(token) == "" {
		return "", errors.New("gemini token is required")
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base url: %w", err)
	}
	apiPath := strings.TrimSuffix(u.Path, "/")
	if !strings.HasSuffix(apiPath, "/v1") && !strings.HasSuffix(apiPath, "/v1beta") {
		apiPath = path.Join(apiPath, "/v1beta")
	}
	verb := "generateContent"
	if stream {
		verb = "streamGenerateContent"
	}
	u.Path = path.Join(apiPath, "models", fmt.Sprintf("%s:%s", model, verb))
	query := u.Query()
	query.Set("key", token)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func readGeminiError(body io.Reader, status int) error {
	var resp geminiGenerateContentResponse
	_ = json.NewDecoder(body).Decode(&resp)
	if resp.Error != nil && resp.Error.Message != "" {
		return fmt.Errorf("gemini request failed: %s (status %d)", resp.Error.Message, status)
	}
	return fmt.Errorf("gemini request failed with status %d", status)
}

func buildGeminiContents(messages []Message) ([]geminiContent, *geminiSystemInstruction) {
	if len(messages) == 0 {
		return nil, nil
	}
	var system *geminiSystemInstruction
	start := 0
	if messages[0].Role == "system" {
		system = &geminiSystemInstruction{
			Parts: []geminiPart{{Text: messages[0].Content}},
		}
		start = 1
	}
	contents := make([]geminiContent, 0, len(messages)-start)
	for _, message := range messages[start:] {
		role := message.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: message.Content}},
		})
	}
	return contents, system
}

func flattenGeminiContent(content geminiContent) string {
	if len(content.Parts) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, part := range content.Parts {
		if part.Text == "" {
			continue
		}
		builder.WriteString(part.Text)
	}
	return builder.String()
}

type geminiGenerateContentRequest struct {
	Contents          []geminiContent          `json:"contents"`
	SystemInstruction *geminiSystemInstruction `json:"systemInstruction,omitempty"`
}

type geminiGenerateContentResponse struct {
	Candidates   []geminiCandidate `json:"candidates"`
	ModelVersion string            `json:"modelVersion,omitempty"`
	Error        *geminiError      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiError struct {
	Message string `json:"message"`
	Status  string `json:"status,omitempty"`
}
