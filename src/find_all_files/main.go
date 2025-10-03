package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bodrovis/lokalise-actions-common/v2/githuboutput"

	"github.com/bodrovis/lokalise-actions-common/v2/parsers"

	"github.com/bmatcuk/doublestar/v4"
)

// This program finds all translation files based on environment configurations.
// It supports both flat and nested directory naming conventions and outputs the list
// of translation files found to GitHub Actions output.

// exitFunc is a function variable that defaults to os.Exit.
// This can be overridden in tests to capture exit behavior.
var exitFunc = os.Exit

func main() {
	// Read and validate environment variables
	translationsPaths, baseLang, fileExt, namePattern := validateEnvironment()

	// Parse flatNaming parameter
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		returnWithError("invalid value for FLAT_NAMING environment variable; expected true or false")
	}

	// Find all translation files based on the provided configurations
	allFiles, err := findAllTranslationFiles(translationsPaths, flatNaming, baseLang, fileExt, namePattern)
	if err != nil {
		returnWithError(fmt.Sprintf("unable to find translation files: %v", err))
	}

	// Process and write `allFiles` to GitHub Actions output
	processAllFiles(allFiles, githuboutput.WriteToGitHubOutput)
}

// validateEnvironment ensures required environment variables are set and parses initial values.
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

	fileExt := os.Getenv("FILE_EXT")
	if fileExt == "" {
		fileExt = os.Getenv("FILE_FORMAT")
	}
	if fileExt == "" {
		returnWithError("Cannot infer file extension. Make sure FILE_FORMAT or FILE_EXT environment variables are set")
	}

	return translationsPaths, baseLang, fileExt, namePattern
}

// processAllFiles writes the found translation files to GitHub Actions output.
func processAllFiles(allFiles []string, writeOutput func(key, value string) bool) {
	if len(allFiles) > 0 {
		// Join all file paths into a comma-separated string
		allFilesStr := strings.Join(allFiles, ",")
		// Write the list of files and set has_files to true
		if !writeOutput("ALL_FILES", allFilesStr) || !writeOutput("has_files", "true") {
			returnWithError("cannot write to GITHUB_OUTPUT")
		}
	} else {
		// No files found, set has_files to false
		if !writeOutput("has_files", "false") {
			returnWithError("cannot write to GITHUB_OUTPUT")
		}
	}
}

// findAllTranslationFiles searches for translation files based on the given paths and naming conventions.
// It supports both flat naming (all translations in one file) and nested directories per language.
func findAllTranslationFiles(paths []string, flatNaming bool, baseLang, fileExt string, namePattern string) ([]string, error) {
	var allFiles []string

	for _, path := range paths {
		if path == "" {
			continue
		}

		if namePattern != "" {
			// Convert to forward slashes and ensure the pattern is relative to the project root
			pattern := filepath.ToSlash(filepath.Join(path, namePattern))

			// Use doublestar.Glob with the project root as the FS
			matches, err := doublestar.Glob(os.DirFS("."), pattern)
			if err != nil {
				return nil, fmt.Errorf("error applying name pattern %s: %v", pattern, err)
			}

			allFiles = append(allFiles, matches...)
		} else if flatNaming {
			// For flat naming, look for a single translation file named as baseLang.fileExt in the path
			targetFile := filepath.Join(path, fmt.Sprintf("%s.%s", baseLang, fileExt))

			info, err := os.Stat(targetFile)
			if err == nil && !info.IsDir() {
				allFiles = append(allFiles, targetFile)
			} else if err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("error accessing file %s: %v", targetFile, err)
			}
		} else {
			// For nested directories, look for a directory named baseLang and search for translation files within it
			targetDir := filepath.Join(path, baseLang)
			if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
				err := filepath.WalkDir(targetDir, func(filePath string, d os.DirEntry, err error) error {
					if err != nil {
						return fmt.Errorf("error walking through directory %s: %v", targetDir, err)
					}
					if !d.IsDir() && strings.HasSuffix(d.Name(), fmt.Sprintf(".%s", fileExt)) {
						allFiles = append(allFiles, filePath)
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

	return allFiles, nil
}

func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
