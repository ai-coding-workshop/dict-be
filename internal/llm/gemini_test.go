package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-test:generateContent" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "token" {
			t.Fatalf("missing api key query")
		}
		var req geminiGenerateContentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Contents) != 1 || req.Contents[0].Role != "user" {
			t.Fatalf("unexpected contents: %+v", req.Contents)
		}
		resp := geminiGenerateContentResponse{
			ModelVersion: "gemini-test",
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role:  "model",
						Parts: []geminiPart{{Text: "hello"}},
					},
					FinishReason: "STOP",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewGeminiClient(GeminiConfig{
		BaseURL: server.URL,
		Token:   "token",
		Model:   "gemini-test",
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
	if resp.FinishReason != "STOP" {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
	if resp.Model != "gemini-test" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
}

func TestGeminiChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-test:streamGenerateContent" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "token" {
			t.Fatalf("missing api key query")
		}
		var req geminiGenerateContentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		chunks := []string{
			`data: {"modelVersion":"gemini-test","candidates":[{"content":{"role":"model","parts":[{"text":"he"}]}}]}` + "\n\n",
			`data: {"candidates":[{"content":{"role":"model","parts":[{"text":"llo"}]},"finishReason":"STOP"}]}` + "\n\n",
		}
		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	client, err := NewGeminiClient(GeminiConfig{
		BaseURL: server.URL,
		Token:   "token",
		Model:   "gemini-test",
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
	if resp.FinishReason != "STOP" {
		t.Fatalf("unexpected finish reason: %s", resp.FinishReason)
	}
	if resp.Model != "gemini-test" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
}
