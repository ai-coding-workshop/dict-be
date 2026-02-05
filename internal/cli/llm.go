package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"dict-be/internal/config"
	"dict-be/internal/llm"

	"github.com/spf13/cobra"
)

type llmChatOptions struct {
	Prompt   string
	System   string
	Stream   bool
	NoStream bool
	Model    string
	URL      string
	Token    string
}

func newLLMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "llm",
		Short: "Interact with LLM providers",
	}

	cmd.AddCommand(newLLMChatCmd())
	cmd.AddCommand(newLLMTestCmd())
	return cmd
}

func newLLMChatCmd() *cobra.Command {
	opts := &llmChatOptions{}
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Send a chat completion request",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLLMChat(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Prompt, "prompt", "", "prompt content (read stdin if empty)")
	cmd.Flags().StringVar(&opts.System, "system", "", "system prompt")
	cmd.Flags().BoolVar(&opts.Stream, "stream", false, "stream response")
	cmd.Flags().BoolVar(&opts.NoStream, "no-stream", false, "disable streaming response")
	cmd.Flags().StringVar(&opts.Model, "model", "", "override model name")
	cmd.Flags().StringVar(&opts.URL, "url", "", "override base url")
	cmd.Flags().StringVar(&opts.Token, "token", "", "override access token")

	return cmd
}

func runLLMChat(cmd *cobra.Command, opts *llmChatOptions) error {
	if opts.Stream && opts.NoStream {
		return errors.New("only one of --stream or --no-stream can be set")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if cfg.LLM.Type == "" {
		cfg.LLM.Type = "openai"
	}
	if cfg.LLM.Type != "openai" {
		return fmt.Errorf("unsupported llm.type: %s", cfg.LLM.Type)
	}

	prompt := strings.TrimSpace(opts.Prompt)
	if prompt == "" {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("read prompt: %w", err)
		}
		prompt = strings.TrimSpace(string(data))
	}
	if prompt == "" {
		return errors.New("prompt is required")
	}

	model := firstNonEmpty(opts.Model, cfg.LLM.Model)
	url := firstNonEmpty(opts.URL, cfg.LLM.URL)
	token := firstNonEmpty(opts.Token, cfg.LLM.Token)

	client, err := llm.NewOpenAIClient(llm.OpenAIConfig{
		BaseURL: url,
		Token:   token,
		Model:   model,
	})
	if err != nil {
		return err
	}

	req := llm.ChatRequest{
		Model:    model,
		Messages: buildMessages(opts.System, prompt),
	}

	if opts.Stream {
		_, err = client.ChatStream(context.Background(), req, func(delta string) error {
			_, writeErr := fmt.Fprint(cmd.OutOrStdout(), delta)
			return writeErr
		})
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		return nil
	}

	resp, err := client.Chat(context.Background(), req)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), resp.Content)
	return err
}

type llmTestOptions struct {
	Stream   bool
	NoStream bool
	Model    string
	URL      string
	Token    string
}

func newLLMTestCmd() *cobra.Command {
	opts := &llmTestOptions{}
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test LLM connectivity with config or flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLLMTest(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Stream, "stream", false, "stream response")
	cmd.Flags().BoolVar(&opts.NoStream, "no-stream", false, "disable streaming response")
	cmd.Flags().StringVar(&opts.Model, "model", "", "override model name")
	cmd.Flags().StringVar(&opts.URL, "url", "", "override base url")
	cmd.Flags().StringVar(&opts.Token, "token", "", "override access token")

	return cmd
}

func runLLMTest(cmd *cobra.Command, opts *llmTestOptions) error {
	if opts.Stream && opts.NoStream {
		return errors.New("only one of --stream or --no-stream can be set")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.LLM.Type == "" {
		cfg.LLM.Type = "openai"
	}
	if cfg.LLM.Type != "openai" {
		return fmt.Errorf("unsupported llm.type: %s", cfg.LLM.Type)
	}

	model := firstNonEmpty(opts.Model, cfg.LLM.Model)
	url := firstNonEmpty(opts.URL, cfg.LLM.URL)
	token := firstNonEmpty(opts.Token, cfg.LLM.Token)

	client, err := llm.NewOpenAIClient(llm.OpenAIConfig{
		BaseURL: url,
		Token:   token,
		Model:   model,
	})
	if err != nil {
		return err
	}

	req := llm.ChatRequest{
		Model: model,
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: "ping",
			},
		},
	}

	if opts.Stream {
		_, err = client.ChatStream(context.Background(), req, func(delta string) error {
			_, writeErr := fmt.Fprint(cmd.OutOrStdout(), delta)
			return writeErr
		})
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		return nil
	}

	resp, err := client.Chat(context.Background(), req)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), resp.Content)
	return err
}

func buildMessages(system, prompt string) []llm.Message {
	messages := make([]llm.Message, 0, 2)
	if strings.TrimSpace(system) != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: system,
		})
	}
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: prompt,
	})
	return messages
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
