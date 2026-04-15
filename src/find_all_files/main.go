package main

import (
	"fmt"
	"os"

	"github.com/bodrovis/lokalise-actions-common/v2/githuboutput"
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
		findAllTranslationFiles,
		processAllFiles,
		githuboutput.WriteToGitHubOutput,
	)
}

type findFunc func([]string, bool, string, []string, string) ([]string, error)

func runWith(
	validate func() (config, error),
	find findFunc,
	process func([]string, func(string, string) bool) error,
	write func(string, string) bool,
) error {
	// Read and validate required env variables.
	cfg, err := validate()
	if err != nil {
		return err
	}

	// Discover files according to the selected strategy.
	allFiles, err := find(
		cfg.Paths,
		cfg.FlatNaming,
		cfg.BaseLang,
		cfg.FileExts,
		cfg.NamePattern,
	)
	if err != nil {
		return fmt.Errorf("unable to find translation files: %w", err)
	}

	// Write outputs for downstream workflow steps.
	if err := process(allFiles, write); err != nil {
		return err
	}

	return nil
}

// returnWithError prints an error and exits with a non-zero code.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
