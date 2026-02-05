package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "token" {
			t.Fatalf("missing api key header")
		}
		var req anthropicChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "claude-test" {
			t.Fatalf("unexpected model: %s", req.Model)
		}
		resp := anthropicChatResponse{
			Model: "claude-test",
			Content: []anthropicContent{
				{Type: "text", Text: "hello"},
			},
			StopReason: "end_turn",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewAnthropicClient(AnthropicConfig{
		BaseURL: server.URL,
		Token:   "token",
		Model:   "claude-test",
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
	if resp.FinishReason != "end_turn" {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
}

func TestAnthropicChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req anthropicChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !req.Stream {
			t.Fatalf("expected stream request")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		chunks := []string{
			`data: {"type":"message_start","message":{"model":"claude-test"}}` + "\n\n",
			`data: {"type":"content_block_delta","delta":{"text":"he"}}` + "\n\n",
			`data: {"type":"content_block_delta","delta":{"text":"llo"}}` + "\n\n",
			`data: {"type":"message_delta","stop_reason":"end_turn"}` + "\n\n",
		}
		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	client, err := NewAnthropicClient(AnthropicConfig{
		BaseURL: server.URL,
		Token:   "token",
		Model:   "claude-test",
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
	if resp.FinishReason != "end_turn" {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
	if resp.Model != "claude-test" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
}
