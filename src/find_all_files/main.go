package main

import (
	"fmt"
	"githuboutput"
	"os"
	"parsepaths"
	"path/filepath"
	"strings"
)

// findAllTranslationFiles finds all translation files based on the environment configuration
func findAllTranslationFiles(paths []string, flatNaming, baseLang, fileFormat string) ([]string, error) {
	var allFiles []string

	for _, path := range paths {
		if path == "" {
			continue
		}

		if flatNaming == "true" {
			// Check for single file with flat naming convention
			targetFile := filepath.Join(path, fmt.Sprintf("%s.%s", baseLang, fileFormat))
			if info, err := os.Stat(targetFile); err == nil && !info.IsDir() {
				allFiles = append(allFiles, targetFile)
			} else if err != nil {
				return nil, fmt.Errorf("error accessing file %s: %v", targetFile, err)
			}
		} else {
			// Check for directory and find files recursively
			targetDir := filepath.Join(path, baseLang)
			if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
				err := filepath.Walk(targetDir, func(filePath string, info os.FileInfo, err error) error {
					if err != nil {
						return fmt.Errorf("error walking through directory %s: %v", targetDir, err)
					}
					if !info.IsDir() && strings.HasSuffix(info.Name(), fmt.Sprintf(".%s", fileFormat)) {
						allFiles = append(allFiles, filePath)
					}
					return nil
				})
				if err != nil {
					return nil, err // Return error encountered during file walk
				}
			} else if err != nil {
				return nil, fmt.Errorf("error accessing directory %s: %v", targetDir, err)
			}
		}
	}

	return allFiles, nil
}

func main() {
	// Read and validate environment variables
	translationsPaths := parsepaths.ParsePaths(os.Getenv("TRANSLATIONS_PATH"))
	flatNaming := os.Getenv("FLAT_NAMING")
	baseLang := os.Getenv("BASE_LANG")
	fileFormat := os.Getenv("FILE_FORMAT")

	if len(translationsPaths) == 0 || baseLang == "" || fileFormat == "" {
		returnWithError("missing required environment variables")
	}

	// Find all translation files
	allFiles, err := findAllTranslationFiles(translationsPaths, flatNaming, baseLang, fileFormat)
	if err != nil {
		returnWithError(fmt.Sprintf("unable to find translation files: %v", err))
	}

	// If files are found, write output to GitHub
	if len(allFiles) > 0 {
		allFilesStr := strings.Join(allFiles, ",")
		if !githuboutput.WriteToGitHubOutput("ALL_FILES", allFilesStr) || !githuboutput.WriteToGitHubOutput("has_files", "true") {
			returnWithError("cannot write to GITHUB_OUTPUT")
		}
	}
}

func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	os.Exit(1)
}
