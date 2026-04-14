package main

import (
	"strings"
)

// processAllFiles emits GitHub Action outputs.
func processAllFiles(allFiles []string, writeOutput func(key, value string) bool) {
	if len(allFiles) == 0 {
		if !writeOutput("has_files", "false") {
			returnWithError("cannot write to GITHUB_OUTPUT")
		}
		return
	}

	if !writeOutput("ALL_FILES", strings.Join(allFiles, ",")) || !writeOutput("has_files", "true") {
		returnWithError("cannot write to GITHUB_OUTPUT")
	}
}
