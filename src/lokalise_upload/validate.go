package main

import (
	"fmt"
	"os"
)

// validate performs input sanity checks before any network calls.
// It fails fast with actionable messages for CI logs.
func validate(cfg UploadConfig) error {
	if err := validateFile(cfg.FilePath); err != nil {
		return err
	}
	if err := validateRequiredFields(cfg); err != nil {
		return err
	}
	if err := validateTaggingInputs(cfg); err != nil {
		return err
	}
	return nil
}

// validateRequiredFields checks the minimum required Lokalise settings.
func validateRequiredFields(cfg UploadConfig) error {
	if cfg.ProjectID == "" {
		return fmt.Errorf("project ID is required and cannot be empty")
	}
	if cfg.Token == "" {
		return fmt.Errorf("API token is required and cannot be empty")
	}
	if cfg.LangISO == "" {
		return fmt.Errorf("base language (BASE_LANG) is required and cannot be empty")
	}
	return nil
}

// validateTaggingInputs ensures branch metadata is available when tagging is enabled.
func validateTaggingInputs(cfg UploadConfig) error {
	if !cfg.SkipTagging && cfg.GitHubRefName == "" {
		return fmt.Errorf("GitHub reference name (GITHUB_HEAD_REF or GITHUB_REF_NAME) is required when tagging is enabled")
	}
	return nil
}

// validateFile ensures the path exists and points to a regular file.
func validateFile(filePath string) error {
	fi, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file %q does not exist", filePath)
	}
	if err != nil {
		return fmt.Errorf("cannot stat file %q: %w", filePath, err)
	}
	if fi.IsDir() {
		return fmt.Errorf("path %q is a directory, not a file", filePath)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("path %q is not a regular file", filePath)
	}
	return nil
}
