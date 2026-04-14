package main

import (
	"fmt"
	"os"
)

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

func main() {
	// Read and validate inputs from the environment.
	// This step makes sure we have enough info to derive a set of pathspecs.
	cfg := validateEnvironment()

	// We persist the generated pathspecs to a file that is later consumed by
	// tj-actions/changed-files via `files_from_source_file`.
	file := createOutputFile()
	defer closeOutputFile(file)

	// Emit one pathspec per line. Consumers expect newline-separated patterns.
	// Each line can be a direct file path or a glob (git pathspec-style).

	if err := storeTranslationPaths(
		cfg.Paths,
		cfg.FlatNaming,
		cfg.BaseLang,
		cfg.FileExts,
		cfg.NamePattern,
		file,
	); err != nil {
		returnWithError(fmt.Sprintf("cannot store translation paths: %v", err))
	}
}

// returnWithError prints an error and exits non-zero.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
