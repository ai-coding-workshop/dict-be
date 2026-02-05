package main

import (
	"os"

	"dict-be/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
