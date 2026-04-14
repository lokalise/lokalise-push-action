package main

import (
	"fmt"
	"os"
)

// validate performs input sanity checks before any network calls.
// It fails fast with actionable messages for CI logs.
func validate(cfg UploadConfig) {
	validateFile(cfg.FilePath)
	validateRequiredFields(cfg)
	validateTaggingInputs(cfg)
}

// validateRequiredFields checks the minimum required Lokalise settings.
func validateRequiredFields(cfg UploadConfig) {
	if cfg.ProjectID == "" {
		returnWithError("Project ID is required and cannot be empty.")
	}
	if cfg.Token == "" {
		returnWithError("API token is required and cannot be empty.")
	}
	if cfg.LangISO == "" {
		returnWithError("Base language (BASE_LANG) is required and cannot be empty.")
	}
}

// validateTaggingInputs ensures branch metadata is available when tagging is enabled.
func validateTaggingInputs(cfg UploadConfig) {
	if !cfg.SkipTagging && cfg.GitHubRefName == "" {
		returnWithError("GitHub reference name (GITHUB_REF_NAME) is required and cannot be empty.")
	}
}

// validateFile ensures the path exists and points to a regular file.
func validateFile(filePath string) {
	fi, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		returnWithError(fmt.Sprintf("File %q does not exist", filePath))
	}
	if err != nil {
		returnWithError(fmt.Sprintf("Cannot stat file %s: %v", filePath, err))
	}
	if fi.IsDir() {
		returnWithError(fmt.Sprintf("Path %s is a directory, not a file.", filePath))
	}
	if !fi.Mode().IsRegular() {
		returnWithError(fmt.Sprintf("Path %s is not a regular file.", filePath))
	}
}
