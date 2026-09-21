package main

import (
	"fmt"
	"os"

	"github.com/bodrovis/lokalise-actions-common/v2/githuboutput"
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

func run() error {
	return runWith(
		validateEnvironment,
		findAllTranslationFiles,
		processAllFiles,
		githuboutput.WriteToGitHubOutput,
	)
}

type (
	validateFunc func() (config, error)
	findFunc     func([]string, bool, string, []string, string, []string) ([]string, error)
	writeFunc    func(string, string) bool
	processFunc  func([]string, writeFunc) error
)

func runWith(
	validate validateFunc,
	find findFunc,
	process processFunc,
	write writeFunc,
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
		cfg.ExcludePatterns,
	)
	if err != nil {
		return fmt.Errorf("unable to find translation files: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d unique files\n", len(allFiles))

	// Write outputs for downstream workflow steps.
	if err := process(allFiles, write); err != nil {
		return err
	}

	return nil
}
