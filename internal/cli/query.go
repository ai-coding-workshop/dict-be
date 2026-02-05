package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newQueryCmd() *cobra.Command {
	var inputFile string
	cmd := &cobra.Command{
		Use:   "query [text...]",
		Short: "Read query from args, file, or stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := readInput(args, inputFile, cmd.InOrStdin())
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), input)
			return nil
		},
	}
	cmd.Flags().StringVarP(&inputFile, "file", "F", "", "query file, use -F- for stdin")
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
