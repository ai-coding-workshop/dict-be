package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "gpt-test" {
			t.Fatalf("unexpected model: %s", req.Model)
		}
		resp := openAIChatResponse{
			Model: "gpt-test",
			Choices: []struct {
				Message      Message `json:"message"`
				Delta        Message `json:"delta"`
				FinishReason string  `json:"finish_reason"`
			}{
				{
					Message: Message{
						Role:    "assistant",
						Content: "hello",
					},
					FinishReason: "stop",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewOpenAIClient(OpenAIConfig{
		BaseURL: server.URL,
		Token:   "token",
		Model:   "gpt-test",
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Content != "hello" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
}

func TestOpenAIChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !req.Stream {
			t.Fatalf("expected stream request")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		chunks := []string{
			`data: {"model":"gpt-test","choices":[{"delta":{"content":"he"}}]}` + "\n\n",
			`data: {"choices":[{"delta":{"content":"llo"},"finish_reason":"stop"}]}` + "\n\n",
			"data: [DONE]\n\n",
		}
		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	client, err := NewOpenAIClient(OpenAIConfig{
		BaseURL: server.URL,
		Token:   "token",
		Model:   "gpt-test",
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var streamed strings.Builder
	resp, err := client.ChatStream(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	}, func(delta string) error {
		streamed.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if streamed.String() != "hello" {
		t.Fatalf("unexpected stream content: %s", streamed.String())
	}
	if resp.Content != "hello" {
		t.Fatalf("unexpected response content: %s", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
}
