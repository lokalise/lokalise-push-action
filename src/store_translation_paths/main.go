package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bodrovis/lokalise-actions-common/parsepaths"
)

// exitFunc is a function variable that defaults to os.Exit.
// This can be overridden in tests to capture exit behavior.
var exitFunc = os.Exit

func main() {
	// Validate and parse environment variables
	translationsPaths, baseLang, fileFormat := validateEnvironment()

	// Parse flat naming
	flatNaming := parseFlatNaming(os.Getenv("FLAT_NAMING"))

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
	if err := storeTranslationPaths(translationsPaths, flatNaming, baseLang, fileFormat, file); err != nil {
		returnWithError(fmt.Sprintf("cannot store translation paths: %v", err))
	}
}

// validateEnvironment checks and retrieves the necessary environment variables.
func validateEnvironment() ([]string, string, string) {
	translationsPaths := parsepaths.ParsePaths(os.Getenv("TRANSLATIONS_PATH"))
	baseLang := os.Getenv("BASE_LANG")
	fileFormat := os.Getenv("FILE_FORMAT")

	if len(translationsPaths) == 0 {
		returnWithError("TRANSLATIONS_PATH is not set or is empty")
	}
	if baseLang == "" {
		returnWithError("BASE_LANG is not set or is empty")
	}
	if fileFormat == "" {
		returnWithError("FILE_FORMAT is not set or is empty")
	}

	return translationsPaths, baseLang, fileFormat
}

// parseFlatNaming parses the FLAT_NAMING environment variable as a boolean.
func parseFlatNaming(flatNamingEnv string) bool {
	if flatNamingEnv == "" {
		return false
	}

	flatNaming, err := strconv.ParseBool(flatNamingEnv)
	if err != nil {
		returnWithError("invalid value for FLAT_NAMING environment variable; expected true or false")
	}

	return flatNaming
}

// storeTranslationPaths generates paths and writes them to paths.txt based on environment variables.
// It constructs the appropriate file paths or glob patterns depending on the naming convention.
func storeTranslationPaths(paths []string, flatNaming bool, baseLang, fileFormat string, writer io.Writer) error {
	for _, path := range paths {
		if path == "" {
			continue // Skip empty paths.
		}

		var formattedPath string
		if flatNaming {
			// For flat naming, construct the path to the base language file.
			// Example: "./path/to/translations/en.json"
			formattedPath = filepath.Join(".", path, fmt.Sprintf("%s.%s", baseLang, fileFormat))
		} else {
			// For nested directories, construct a glob pattern to match all files in the base language directory.
			// Example: "./path/to/translations/en/**/*.json"
			formattedPath = filepath.Join(".", path, baseLang, "**", fmt.Sprintf("*.%s", fileFormat))
		}

		// Write the formatted path to the file, adding a newline character.
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
