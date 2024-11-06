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
func findAllTranslationFiles(paths []string, flatNaming, baseLang, fileFormat string) ([]string, bool) {
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
			}
		} else {
			// Check for directory and find files recursively
			targetDir := filepath.Join(path, baseLang)
			if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
				err := filepath.Walk(targetDir, func(filePath string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && strings.HasSuffix(info.Name(), fmt.Sprintf(".%s", fileFormat)) {
						allFiles = append(allFiles, filePath)
					}
					return nil
				})
				if err != nil {
					return nil, false
				}
			}
		}
	}

	return allFiles, true
}

func main() {
	// Read environment variables
	translationsPaths := parsepaths.ParsePaths(os.Getenv("TRANSLATIONS_PATH"))
	flatNaming := os.Getenv("FLAT_NAMING")
	baseLang := os.Getenv("BASE_LANG")
	fileFormat := os.Getenv("FILE_FORMAT")

	// Find all translation files
	allFiles, ok := findAllTranslationFiles(translationsPaths, flatNaming, baseLang, fileFormat)
	if !ok {
		os.Exit(1)
	}

	if len(allFiles) > 0 {
		allFilesStr := strings.Join(allFiles, ",")

		if !githuboutput.WriteToGitHubOutput("ALL_FILES", allFilesStr) || !githuboutput.WriteToGitHubOutput("has_files", "true") {
			os.Exit(1)
		}
	}
}
