package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bodrovis/lokalise-actions-common/parsepaths"
)

// storeTranslationPaths generates paths and writes them to paths.txt based on environment variables.
// It constructs the appropriate file paths or glob patterns depending on the naming convention.
func storeTranslationPaths(paths []string, flatNaming bool, baseLang, fileFormat string) error {
	// Create (or overwrite) the txt file to store the generated paths.
	file, err := os.Create("lok_action_paths_temp.txt")
	if err != nil {
		return err
	}
	defer file.Close()

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
		if _, err := file.WriteString(formattedPath + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// Read environment variables and parse the translations paths.
	translationsPaths := parsepaths.ParsePaths(os.Getenv("TRANSLATIONS_PATH"))
	flatNamingEnv := os.Getenv("FLAT_NAMING")
	baseLang := os.Getenv("BASE_LANG")
	fileFormat := os.Getenv("FILE_FORMAT")

	// Validate that the required environment variables are set.
	if len(translationsPaths) == 0 || baseLang == "" || fileFormat == "" {
		fmt.Fprintln(os.Stderr, "Error: missing required environment variables")
		os.Exit(1)
	}

	// Parse FLAT_NAMING environment variable as a boolean.
	flatNaming := false
	if flatNamingEnv != "" {
		var err error
		flatNaming, err = strconv.ParseBool(flatNamingEnv)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: invalid value for FLAT_NAMING environment variable; expected true or false")
			os.Exit(1)
		}
	}

	// Generate and store the translation paths.
	if err := storeTranslationPaths(translationsPaths, flatNaming, baseLang, fileFormat); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
