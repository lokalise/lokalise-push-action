package main

import (
	"fmt"
	"os"

	"github.com/bodrovis/lokalise-actions-common/v2/normalizers"
	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
)

type config struct {
	Paths       []string
	BaseLang    string
	FileExts    []string
	NamePattern string
	FlatNaming  bool
}

// validateEnvironment enforces presence of required inputs and normalizes them.
func validateEnvironment() config {
	return config{
		Paths:       parseTranslationsPaths(),
		BaseLang:    parseBaseLang(),
		FileExts:    parseFileExtensions(),
		NamePattern: parseNamePattern(),
		FlatNaming:  parseFlatNaming(),
	}
}

func parseNamePattern() string {
	namePattern, err := normalizers.NormalizeOptionalNamePattern(os.Getenv("NAME_PATTERN"))
	if err != nil {
		returnWithError(err.Error())
	}
	return namePattern
}

func parseFlatNaming() bool {
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		returnWithError("invalid FLAT_NAMING: expected true or false")
	}
	return flatNaming
}

func parseFileExtensions() []string {
	fileExts, err := normalizers.NormalizeFileExtensions(parsers.ParseStringArrayEnv("FILE_EXT"))
	if err != nil {
		returnWithError(fmt.Sprintf("invalid FILE_EXT: %v", err))
	}
	return fileExts
}

// parseTranslationsPaths parses and validates repo-relative translation roots.
func parseTranslationsPaths() []string {
	paths, err := parsers.ParseRepoRelativePathsEnv("TRANSLATIONS_PATH")
	if err != nil {
		returnWithError(fmt.Sprintf("failed to process params: %v", err))
	}
	return paths
}

// parseBaseLang returns the configured base language or fails if missing.
func parseBaseLang() string {
	baseLang := os.Getenv("BASE_LANG")
	if baseLang == "" {
		returnWithError("BASE_LANG is not set or is empty")
	}
	return baseLang
}
