package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
)

// exitFunc is a function variable that defaults to os.Exit.
// This can be overridden in tests to capture exit behavior.
var exitFunc = os.Exit

func main() {
	// Validate and parse environment variables
	translationsPaths, baseLang, fileFormat, namePattern := validateEnvironment()

	// Parse flat naming
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		returnWithError("invalid value for FLAT_NAMING environment variable; expected true or false")
	}

	// Open the output file
	file, err := os.Create("lok_action_paths_temp.txt")
	if err != nil {
		returnWithError(fmt.Sprintf("cannot create output file: %v", err))
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file properly: %v\n", cerr)
		}
	}()

	// Generate and store the translation paths
	if err := storeTranslationPaths(translationsPaths, flatNaming, baseLang, fileFormat, namePattern, file); err != nil {
		returnWithError(fmt.Sprintf("cannot store translation paths: %v", err))
	}
}

// validateEnvironment checks and retrieves the necessary environment variables.
func validateEnvironment() ([]string, string, string, string) {
	translationsPaths := parsers.ParseStringArrayEnv("TRANSLATIONS_PATH")
	baseLang := os.Getenv("BASE_LANG")
	fileFormat := os.Getenv("FILE_FORMAT")
	namePattern := os.Getenv("NAME_PATTERN")

	if len(translationsPaths) == 0 {
		returnWithError("TRANSLATIONS_PATH is not set or is empty")
	}
	if baseLang == "" {
		returnWithError("BASE_LANG is not set or is empty")
	}
	if fileFormat == "" {
		returnWithError("FILE_FORMAT is not set or is empty")
	}

	return translationsPaths, baseLang, fileFormat, namePattern
}

// storeTranslationPaths generates paths and writes them to paths.txt based on environment variables.
// It constructs the appropriate file paths or glob patterns depending on the naming convention and name pattern.
func storeTranslationPaths(paths []string, flatNaming bool, baseLang, fileFormat, namePattern string, writer io.Writer) error {
	for _, path := range paths {
		if path == "" {
			continue // Skip empty paths.
		}

		var formattedPath string
		if namePattern != "" {
			// Use the custom name pattern provided by the user.
			formattedPath = filepath.Join(".", path, namePattern)
		} else if flatNaming {
			// For flat naming, construct the path to the base language file.
			formattedPath = filepath.Join(".", path, fmt.Sprintf("%s.%s", baseLang, fileFormat))
		} else {
			// For nested directories, construct a glob pattern to match all files in the base language directory.
			formattedPath = filepath.Join(".", path, baseLang, "**", fmt.Sprintf("*.%s", fileFormat))
		}

		if _, err := writer.Write([]byte(formattedPath + "\n")); err != nil {
			return err
		}
	}

	return nil
}

func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
