package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// writeUniqueLine writes a normalized newline-terminated pathspec once.
func writeUniqueLine(writer io.Writer, seen map[string]struct{}, pathspec string) error {
	line := filepath.ToSlash(filepath.Clean(pathspec))
	if line == "." || line == "" {
		return fmt.Errorf("empty pathspec")
	}

	if _, ok := seen[line]; ok {
		return nil
	}

	if _, err := fmt.Fprintln(writer, line); err != nil {
		return err
	}

	seen[line] = struct{}{}
	return nil
}

// createOutputFile creates the temp file consumed later by changed-files.
func createOutputFile() (*os.File, error) {
	dir := filepath.Join(".git", "lokalise-action")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create output directory: %w", err)
	}

	file, err := os.Create(filepath.Join(dir, "paths.txt"))
	if err != nil {
		return nil, err
	}

	return file, nil
}

// closeOutputFile closes the output file.
func closeOutputFile(file *os.File) error {
	return file.Close()
}
