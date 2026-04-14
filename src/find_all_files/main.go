package main

import (
	"fmt"
	"os"

	"github.com/bodrovis/lokalise-actions-common/v2/githuboutput"
)

// This program discovers translation files based on env configuration.
// It supports two layout styles:
//   - flat:   <root>/<baseLang>.<ext>
//   - nested: <root>/<baseLang>/**/<anything>.<ext>
//
// Optionally, a custom NAME_PATTERN can override both layouts.
// Results are exported via GitHub Actions outputs:
//   - ALL_FILES: comma-separated file list
//   - has_files: true/false
//
// Note: ALL_FILES is comma-separated for downstream shell processing.
// This means file names containing commas are not safely representable
// in that output format.

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

func main() {
	// Read and validate required env variables.
	cfg := validateEnvironment()

	// Discover files according to the selected strategy.
	allFiles, err := findAllTranslationFiles(
		cfg.Paths,
		cfg.FlatNaming,
		cfg.BaseLang,
		cfg.FileExts,
		cfg.NamePattern,
	)
	if err != nil {
		returnWithError(fmt.Sprintf("unable to find translation files: %v", err))
	}

	// Write outputs for downstream workflow steps.
	processAllFiles(allFiles, githuboutput.WriteToGitHubOutput)
}

// returnWithError prints an error and exits with a non-zero code.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
