package main

import (
	"fmt"
	"strings"
)

// processAllFiles emits GitHub Action outputs.
func processAllFiles(allFiles []string, writeOutput func(key, value string) bool) error {
	if len(allFiles) == 0 {
		if !writeOutput("has_files", "false") {
			return fmt.Errorf("cannot write has_files to GITHUB_OUTPUT")
		}
		return nil
	}

	if !writeOutput("ALL_FILES", strings.Join(allFiles, ",")) {
		return fmt.Errorf("cannot write ALL_FILES to GITHUB_OUTPUT")
	}

	if !writeOutput("has_files", "true") {
		return fmt.Errorf("cannot write has_files to GITHUB_OUTPUT")
	}

	return nil
}
