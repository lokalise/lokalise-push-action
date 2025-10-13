package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
)

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

func main() {
	// Read and validate inputs from the environment.
	// This step makes sure we have enough info to derive a set of pathspecs.
	translationsPaths, baseLang, fileExt, namePattern := validateEnvironment()

	// FLAT_NAMING determines whether translations are flat (e.g., locales/en.json)
	// or nested by language (e.g., locales/en/**/*.json).
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		returnWithError("invalid value for FLAT_NAMING environment variable; expected true or false")
	}

	// We persist the generated pathspecs to a file that is later consumed by
	// tj-actions/changed-files via `files_from_source_file`.
	file, err := os.Create("lok_action_paths_temp.txt")
	if err != nil {
		returnWithError(fmt.Sprintf("cannot create output file: %v", err))
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			// Non-fatal warning; the action will likely still succeed if writes completed.
			fmt.Fprintf(os.Stderr, "Warning: failed to close file properly: %v\n", cerr)
		}
	}()

	// Emit one pathspec per line. Consumers expect newline-separated patterns.
	// Each line can be a direct file path or a glob (git pathspec-style).
	if err := storeTranslationPaths(translationsPaths, flatNaming, baseLang, fileExt, namePattern, file); err != nil {
		returnWithError(fmt.Sprintf("cannot store translation paths: %v", err))
	}
}

// validateEnvironment reads required variables and applies simple inference.
// Returns: (paths, base language code, file extension, optional custom name pattern).
func validateEnvironment() ([]string, string, string, string) {
	translationsPaths := parsers.ParseStringArrayEnv("TRANSLATIONS_PATH")
	baseLang := os.Getenv("BASE_LANG")
	namePattern := os.Getenv("NAME_PATTERN")

	if len(translationsPaths) == 0 {
		returnWithError("TRANSLATIONS_PATH is not set or is empty")
	}
	if baseLang == "" {
		returnWithError("BASE_LANG is not set or is empty")
	}

	// FILE_EXT may be provided explicitly; otherwise we fall back to FILE_FORMAT.
	// Note: we do not normalize here (e.g., trimming leading dot). Downstream
	// code should be tolerant, but normalization could make matching more robust.
	fileExt := os.Getenv("FILE_EXT")
	if fileExt == "" {
		fileExt = os.Getenv("FILE_FORMAT")
	}
	if fileExt == "" {
		returnWithError("Cannot infer file extension. Make sure FILE_FORMAT or FILE_EXT environment variables are set")
	}

	return translationsPaths, baseLang, fileExt, namePattern
}

// storeTranslationPaths writes one pathspec per input path, based on layout rules:
// - With NAME_PATTERN: use it as-is under each root (caller is responsible for including filename/ext or globs).
// - Flat naming: a single file per root, e.g. "./locales/en.json".
// - Nested naming: a glob matching all files for base lang, e.g. "./locales/en/**/*.json".
func storeTranslationPaths(paths []string, flatNaming bool, baseLang, fileExt, namePattern string, writer io.Writer) error {
	for _, path := range paths {
		if path == "" {
			continue
		}

		var formattedPath string
		if namePattern != "" {
			// Custom pattern fully overrides default construction.
			// It may be a static file name (e.g., "custom.json") or a glob (e.g., "**/*.yaml").
			formattedPath = filepath.Join(".", path, namePattern)
		} else if flatNaming {
			// Flat layout: one file per language at the root: "<root>/<lang>.<ext>"
			formattedPath = filepath.Join(".", path, fmt.Sprintf("%s.%s", baseLang, fileExt))
		} else {
			// Nested layout: include everything for the base language under its subdirectory.
			// Example: "./locales/en/**/*.json"
			formattedPath = filepath.Join(".", path, baseLang, "**", fmt.Sprintf("*.%s", fileExt))
		}

		// Normalize to forward slashes for cross-platform consistency.
		normalizedPath := filepath.ToSlash(formattedPath)

		// Write one pattern per line: newline-separated list is what the consumer expects.
		if _, err := writer.Write([]byte(normalizedPath + "\n")); err != nil {
			return err
		}
	}

	return nil
}

// returnWithError prints an error and exits non-zero.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
