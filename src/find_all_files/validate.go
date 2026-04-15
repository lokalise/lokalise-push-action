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
func validateEnvironment() (config, error) {
	paths, err := parseTranslationsPaths()
	if err != nil {
		return config{}, err
	}

	baseLang, err := parseBaseLang()
	if err != nil {
		return config{}, err
	}

	fileExts, err := parseFileExtensions()
	if err != nil {
		return config{}, err
	}

	namePattern, err := parseNamePattern()
	if err != nil {
		return config{}, err
	}

	flatNaming, err := parseFlatNaming()
	if err != nil {
		return config{}, err
	}

	return config{
		Paths:       paths,
		BaseLang:    baseLang,
		FileExts:    fileExts,
		NamePattern: namePattern,
		FlatNaming:  flatNaming,
	}, nil
}

func parseNamePattern() (string, error) {
	return normalizers.NormalizeOptionalNamePattern(os.Getenv("NAME_PATTERN"))
}

func parseFlatNaming() (bool, error) {
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		return false, fmt.Errorf("invalid FLAT_NAMING: expected true or false: %w", err)
	}
	return flatNaming, nil
}

func parseFileExtensions() ([]string, error) {
	fileExts, err := normalizers.NormalizeFileExtensions(parsers.ParseStringArrayEnv("FILE_EXT"))
	if err != nil {
		return nil, fmt.Errorf("invalid FILE_EXT: %w", err)
	}
	return fileExts, nil
}

// parseTranslationsPaths parses and validates repo-relative translation roots.
func parseTranslationsPaths() ([]string, error) {
	paths, err := parsers.ParseRepoRelativePathsEnv("TRANSLATIONS_PATH")
	if err != nil {
		return nil, fmt.Errorf("failed to process params: %w", err)
	}
	return paths, nil
}

// parseBaseLang returns the configured base language or fails if missing.
func parseBaseLang() (string, error) {
	baseLang := os.Getenv("BASE_LANG")
	if baseLang == "" {
		return "", fmt.Errorf("BASE_LANG is not set or is empty")
	}
	return baseLang, nil
}
