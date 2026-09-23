package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}

	return 0
}

func run() error {
	return runWith(
		validateEnvironment,
		createOutputFile,
		storeTranslationPaths,
		storeExcludedPaths,
		(*os.File).Close,
	)
}

func runWith(
	validate func() (envConfig, error),
	createFile func(string) (*os.File, error),
	storePaths storePathsFunc,
	storeExcluded storePathsFunc,
	closeFile func(*os.File) error,
) (err error) {
	// Read and validate inputs from the environment.
	cfg, err := validate()
	if err != nil {
		return err
	}

	// Include pathspecs consumed by changed-files via files_from_source_file.
	pathsFile, err := createFile(pathsFileName)
	if err != nil {
		return fmt.Errorf("cannot create translation paths output file: %w", err)
	}

	defer func() {
		if closeErr := closeFile(pathsFile); closeErr != nil {
			err = errors.Join(
				err,
				fmt.Errorf("cannot close translation paths output file: %w", closeErr),
			)
		}
	}()

	// Exclude pathspecs consumed by changed-files via
	// files_ignore_from_source_file.
	excludePathsFile, err := createFile(excludePathsFileName)
	if err != nil {
		return fmt.Errorf("cannot create excluded paths output file: %w", err)
	}

	defer func() {
		if closeErr := closeFile(excludePathsFile); closeErr != nil {
			err = errors.Join(
				err,
				fmt.Errorf("cannot close excluded paths output file: %w", closeErr),
			)
		}
	}()

	// Emit include pathspecs.
	if err := storePaths(cfg, pathsFile); err != nil {
		return fmt.Errorf("cannot store translation paths: %w", err)
	}

	// Emit exclude pathspecs. The output file is created even when there are
	// no exclude patterns, so the workflow can reference it unconditionally.
	if err := storeExcluded(cfg, excludePathsFile); err != nil {
		return fmt.Errorf("cannot store excluded paths: %w", err)
	}

	return nil
}
