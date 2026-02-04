package cli

import (
	"fmt"
	"io"
	"strings"

	"dict-be/internal/version"
)

type Command struct {
	Name        string
	Description string
	Run         func(ctx Context) int
}

type Context struct {
	Args []string
	Out  io.Writer
	Err  io.Writer
}

func Execute(args []string, out io.Writer, errOut io.Writer) int {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	commands := []Command{
		{
			Name:        "help",
			Description: "Show available commands",
			Run: func(ctx Context) int {
				printUsage(ctx.Out, commands)
				return 0
			},
		},
		{
			Name:        "version",
			Description: "Print build version",
			Run: func(ctx Context) int {
				fmt.Fprintln(ctx.Out, version.Version)
				return 0
			},
		},
	}

	if len(args) == 0 {
		printUsage(out, commands)
		return 0
	}

	name := strings.TrimSpace(args[0])
	for _, command := range commands {
		if command.Name == name {
			return command.Run(Context{
				Args: args[1:],
				Out:  out,
				Err:  errOut,
			})
		}
	}

	fmt.Fprintf(errOut, "unknown command: %s\n", name)
	printUsage(errOut, commands)
	return 1
}

func printUsage(out io.Writer, commands []Command) {
	fmt.Fprintln(out, "dict-be - CLI starter")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  dict-be <command> [args]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Commands:")
	for _, command := range commands {
		fmt.Fprintf(out, "  %-10s %s\n", command.Name, command.Description)
	}
}
