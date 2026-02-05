package cli

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"dict-be/internal/config"
	"dict-be/internal/llm"

	"github.com/spf13/cobra"
)

//go:embed query_system.md
//go:embed query_user.md
var queryPromptFS embed.FS

const (
	querySystemPromptPath = "query_system.md"
	queryUserPromptPath   = "query_user.md"
)

type queryOptions struct {
	InputFile      string
	InputLanguage  string
	OutputLanguage string
	Stream         bool
	NoStream       bool
}

func newQueryCmd() *cobra.Command {
	opts := &queryOptions{}
	cmd := &cobra.Command{
		Use:   "query [text...]",
		Short: "Translate query between languages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cmd, opts, args)
		},
	}
	cmd.Flags().StringVarP(&opts.InputFile, "file", "F", "", "query file, use -F- for stdin")
	cmd.Flags().StringVar(&opts.InputLanguage, "input-language", "auto", "input language")
	cmd.Flags().StringVar(&opts.InputLanguage, "in", "auto", "input language")
	cmd.Flags().StringVar(&opts.OutputLanguage, "output-language", "auto", "output language")
	cmd.Flags().StringVar(&opts.OutputLanguage, "out", "auto", "output language")
	cmd.Flags().BoolVar(&opts.Stream, "stream", false, "stream response")
	cmd.Flags().BoolVar(&opts.NoStream, "no-stream", false, "disable streaming response")
	return cmd
}

func runQuery(cmd *cobra.Command, opts *queryOptions, args []string) error {
	if opts.Stream && opts.NoStream {
		return fmt.Errorf("only one of --stream or --no-stream can be set")
	}
	input, err := readInput(args, opts.InputFile, cmd.InOrStdin())
	if err != nil {
		return err
	}
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("input is required")
	}
	inputLanguage, outputLanguage := resolveLanguages(input, opts.InputLanguage, opts.OutputLanguage)
	systemPrompt, userPrompt, err := buildQueryPrompts(input, inputLanguage, outputLanguage)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.LLM.Type == "" {
		cfg.LLM.Type = "openai"
	}

	client, err := newLLMClient(cfg.LLM.Type, cfg.LLM.URL, cfg.LLM.Token, cfg.LLM.Model)
	if err != nil {
		return err
	}

	req := llm.ChatRequest{
		Model:    cfg.LLM.Model,
		Messages: buildMessages(systemPrompt, userPrompt),
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

func readInput(args []string, inputFile string, stdin io.Reader) (string, error) {
	if inputFile != "" && len(args) > 0 {
		return "", fmt.Errorf("input args and -F are mutually exclusive")
	}
	if inputFile == "" {
		if len(args) == 0 {
			return "", fmt.Errorf("missing input: provide args or -F")
		}
		return strings.Join(args, " "), nil
	}
	if inputFile == "-" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return trimTrailingNewline(string(data)), nil
	}
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return trimTrailingNewline(string(data)), nil
}

func trimTrailingNewline(value string) string {
	return strings.TrimRight(value, "\r\n")
}

func buildQueryPrompts(input, inputLanguage, outputLanguage string) (string, string, error) {
	systemTemplate, err := loadQueryPrompt(querySystemPromptPath)
	if err != nil {
		return "", "", err
	}
	userTemplate, err := loadQueryPrompt(queryUserPromptPath)
	if err != nil {
		return "", "", err
	}

	systemPrompt := renderQueryPrompt(systemTemplate, input, inputLanguage, outputLanguage)
	userPrompt := renderQueryPrompt(userTemplate, input, inputLanguage, outputLanguage)
	return systemPrompt, userPrompt, nil
}

func loadQueryPrompt(path string) (string, error) {
	data, err := queryPromptFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt template %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func renderQueryPrompt(template, input, inputLanguage, outputLanguage string) string {
	replacer := strings.NewReplacer(
		"{{input}}", input,
		"{{input_language}}", inputLanguage,
		"{{output_language}}", outputLanguage,
	)
	return replacer.Replace(template)
}

func resolveLanguages(input, inputLanguage, outputLanguage string) (string, string) {
	if inputLanguage == "auto" && outputLanguage == "auto" {
		if containsChinese(input) {
			return "Simplified Chinese", "English"
		}
		return "English", "Simplified Chinese"
	}
	return inputLanguage, outputLanguage
}

func containsChinese(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
