package main

import (
	"fmt"
	"os"

	"github.com/bodrovis/lokalise-actions-common/v2/normalizers"
	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
)

type envConfig struct {
	Paths       []string
	BaseLang    string
	FileExts    []string
	NamePattern string
	FlatNaming  bool
}

// validateEnvironment reads required variables and applies simple inference.
// Returns: (paths, base language code, file extensions, optional custom name pattern).
func validateEnvironment() envConfig {
	return envConfig{
		Paths:       parseTranslationsPaths(),
		BaseLang:    parseBaseLang(),
		FileExts:    parseFileExtensions(),
		NamePattern: parseNamePattern(),
		FlatNaming:  parseFlatNaming(),
	}
}

func parseTranslationsPaths() []string {
	paths, err := parsers.ParseRepoRelativePathsEnv("TRANSLATIONS_PATH")
	if err != nil {
		returnWithError(fmt.Sprintf("invalid TRANSLATIONS_PATH: %v", err))
	}
	return paths
}

func parseBaseLang() string {
	baseLang := os.Getenv("BASE_LANG")
	if baseLang == "" {
		returnWithError("BASE_LANG is not set or is empty")
	}
	return baseLang
}

func parseNamePattern() string {
	namePattern, err := normalizers.NormalizeOptionalNamePattern(os.Getenv("NAME_PATTERN"))
	if err != nil {
		returnWithError(err.Error())
	}
	return namePattern
}

func parseFileExtensions() []string {
	fileExts, err := normalizers.NormalizeFileExtensions(parsers.ParseStringArrayEnv("FILE_EXT"))
	if err != nil {
		returnWithError(fmt.Sprintf("invalid FILE_EXT: %v", err))
	}
	return fileExts
}

func parseFlatNaming() bool {
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		returnWithError("invalid FLAT_NAMING: expected true or false")
	}
	return flatNaming
}
