package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// writeUniqueLine writes a normalized newline-terminated pathspec once.
func writeUniqueLine(writer io.Writer, seen map[string]struct{}, path string) error {
	line := filepath.ToSlash(filepath.Join(".", path))

	if _, ok := seen[line]; ok {
		return nil
	}
	seen[line] = struct{}{}

	_, err := writer.Write([]byte(line + "\n"))
	return err
}

// createOutputFile creates the temp file consumed later by changed-files.
func createOutputFile() (*os.File, error) {
	file, err := os.Create("lok_action_paths_temp.txt")
	if err != nil {
		return nil, fmt.Errorf("cannot create output file: %w", err)
	}

	return file, nil
}

// closeOutputFile closes the output file and prints a warning on failure.
// This warning is non-fatal because the file may have already been written successfully.
func closeOutputFile(file *os.File) {
	if cerr := file.Close(); cerr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close file properly: %v\n", cerr)
	}
}
