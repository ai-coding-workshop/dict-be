package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadQueryFromArgs(t *testing.T) {
	input, err := readInput([]string{"hello", "world"}, "", strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input != "hello world" {
		t.Fatalf("unexpected input: %q", input)
	}
}

func TestReadQueryFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(path, []byte("file input\n"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	input, err := readInput(nil, path, strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input != "file input" {
		t.Fatalf("unexpected input: %q", input)
	}
}

func TestReadQueryFromStdin(t *testing.T) {
	input, err := readInput(nil, "-", strings.NewReader("stdin input\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input != "stdin input" {
		t.Fatalf("unexpected input: %q", input)
	}
}

func TestReadQueryMissing(t *testing.T) {
	_, err := readInput(nil, "", strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestReadQueryConflict(t *testing.T) {
	_, err := readInput([]string{"hello"}, "input.txt", strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected error")
	}
}
