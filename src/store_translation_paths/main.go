package main

import (
	"fmt"
	"os"
)

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

func main() {
	if err := run(); err != nil {
		returnWithError(err.Error())
	}
}

func run() error {
	return runWith(
		validateEnvironment,
		createOutputFile,
		storeTranslationPaths,
		closeOutputFile,
	)
}

func runWith(
	validate func() (envConfig, error),
	createFile func() (*os.File, error),
	store storePathsFunc,
	closeFile func(*os.File),
) error {
	// Read and validate inputs from the environment.
	cfg, err := validate()
	if err != nil {
		return err
	}

	// We persist the generated pathspecs to a file that is later consumed by
	// tj-actions/changed-files via `files_from_source_file`.
	file, err := createFile()
	if err != nil {
		return fmt.Errorf("cannot create output file: %w", err)
	}
	defer closeFile(file)

	// Emit one pathspec per line. Consumers expect newline-separated patterns.
	// Each line can be a direct file path or a glob (git pathspec-style).
	if err := store(cfg, file); err != nil {
		return fmt.Errorf("cannot store translation paths: %w", err)
	}

	return nil
}

// returnWithError prints an error and exits non-zero.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
