package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

func newQueryCmd() *cobra.Command {
	var inputFile string
	var inputLanguage string
	var outputLanguage string
	cmd := &cobra.Command{
		Use:   "query [text...]",
		Short: "Read query from args, file, or stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := readInput(args, inputFile, cmd.InOrStdin())
			if err != nil {
				return err
			}
			inputLanguage, outputLanguage = resolveLanguages(input, inputLanguage, outputLanguage)
			fmt.Fprintln(cmd.OutOrStdout(), input)
			return nil
		},
	}
	cmd.Flags().StringVarP(&inputFile, "file", "F", "", "query file, use -F- for stdin")
	cmd.Flags().StringVar(&inputLanguage, "input-language", "auto", "input language")
	cmd.Flags().StringVar(&outputLanguage, "output-language", "auto", "output language")
	return cmd
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
