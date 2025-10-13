package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/bodrovis/lokalise-actions-common/v2/githuboutput"
	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
)

// This program discovers translation files based on env configuration.
// It supports two layout styles:
//   - flat:   <root>/<baseLang>.<ext>
//   - nested: <root>/<baseLang>/**/<anything>.<ext>
// Optionally, a custom NAME_PATTERN can override both.
// Results are exported via GitHub Actions outputs: ALL_FILES (comma-separated) and has_files (true/false).

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

func main() {
	// Read and validate required env variables.
	translationsPaths, baseLang, fileExt, namePattern := validateEnvironment()

	// Parse FLAT_NAMING: true -> flat files at root; false -> nested per-language directories.
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		returnWithError("invalid value for FLAT_NAMING environment variable; expected true or false")
	}

	// Discover files according to the selected strategy.
	allFiles, err := findAllTranslationFiles(translationsPaths, flatNaming, baseLang, fileExt, namePattern)
	if err != nil {
		returnWithError(fmt.Sprintf("unable to find translation files: %v", err))
	}

	// Write outputs for downstream steps.
	processAllFiles(allFiles, githuboutput.WriteToGitHubOutput)
}

// validateEnvironment enforces presence of required inputs and performs simple inference (FILE_EXT ← FILE_FORMAT).
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

	// FILE_EXT takes precedence; fall back to FILE_FORMAT for convenience.
	fileExt := os.Getenv("FILE_EXT")
	if fileExt == "" {
		fileExt = os.Getenv("FILE_FORMAT")
	}
	if fileExt == "" {
		returnWithError("Cannot infer file extension. Make sure FILE_FORMAT or FILE_EXT environment variables are set")
	}

	return translationsPaths, baseLang, fileExt, namePattern
}

// processAllFiles emits GitHub Action outputs.
// Note: ALL_FILES is a comma-separated list (consumers must handle paths with spaces properly).
func processAllFiles(allFiles []string, writeOutput func(key, value string) bool) {
	if len(allFiles) > 0 {
		allFilesStr := strings.Join(allFiles, ",")
		if !writeOutput("ALL_FILES", allFilesStr) || !writeOutput("has_files", "true") {
			returnWithError("cannot write to GITHUB_OUTPUT")
		}
	} else {
		if !writeOutput("has_files", "false") {
			returnWithError("cannot write to GITHUB_OUTPUT")
		}
	}
}

// findAllTranslationFiles scans each configured root using the chosen strategy.
// - NAME_PATTERN (if provided) overrides layout rules and is treated as a glob under the root.
// - Flat: single file "<root>/<baseLang>.<ext>" if present.
// - Nested: walk "<root>/<baseLang>" and collect files ending with ".<ext>".
func findAllTranslationFiles(paths []string, flatNaming bool, baseLang, fileExt string, namePattern string) ([]string, error) {
	var allFiles []string
	seen := make(map[string]struct{}) // dedup set

	addFile := func(p string) {
		p = filepath.ToSlash(p)
		if _, exists := seen[p]; !exists {
			seen[p] = struct{}{}
			allFiles = append(allFiles, p)
		}
	}

	for _, path := range paths {
		if path == "" {
			continue
		}

		if namePattern != "" {
			// Custom pattern overrides defaults
			pattern := filepath.ToSlash(filepath.Join(path, namePattern))

			matches, err := doublestar.Glob(os.DirFS("."), pattern)
			if err != nil {
				return nil, fmt.Errorf("error applying name pattern %s: %v", pattern, err)
			}
			for _, m := range matches {
				addFile(m)
			}

		} else if flatNaming {
			targetFile := filepath.Join(path, fmt.Sprintf("%s.%s", baseLang, fileExt))
			if info, err := os.Stat(targetFile); err == nil && !info.IsDir() {
				addFile(targetFile)
			} else if err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("error accessing file %s: %v", targetFile, err)
			}

		} else {
			targetDir := filepath.Join(path, baseLang)
			if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
				err := filepath.WalkDir(targetDir, func(filePath string, d os.DirEntry, err error) error {
					if err != nil {
						return fmt.Errorf("error walking through directory %s: %v", targetDir, err)
					}
					if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), fmt.Sprintf(".%s", strings.ToLower(fileExt))) {
						addFile(filePath)
					}
					return nil
				})
				if err != nil {
					return nil, err
				}
			} else if err != nil {
				if !os.IsNotExist(err) {
					return nil, fmt.Errorf("error accessing directory %s: %v", targetDir, err)
				}
			}
		}
	}

	// Sort deterministically for reproducible output
	sort.Strings(allFiles)

	fmt.Fprintf(os.Stderr, "Found %d unique files\n", len(allFiles))

	return allFiles, nil
}

func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
