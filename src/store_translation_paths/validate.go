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
func validateEnvironment() (envConfig, error) {
	paths, err := parseTranslationsPaths()
	if err != nil {
		return envConfig{}, err
	}

	baseLang, err := parseBaseLang()
	if err != nil {
		return envConfig{}, err
	}

	fileExts, err := parseFileExtensions()
	if err != nil {
		return envConfig{}, err
	}

	namePattern, err := parseNamePattern()
	if err != nil {
		return envConfig{}, err
	}

	flatNaming, err := parseFlatNaming()
	if err != nil {
		return envConfig{}, err
	}

	return envConfig{
		Paths:       paths,
		BaseLang:    baseLang,
		FileExts:    fileExts,
		NamePattern: namePattern,
		FlatNaming:  flatNaming,
	}, nil
}

func parseTranslationsPaths() ([]string, error) {
	paths, err := parsers.ParseRepoRelativePathsEnv("TRANSLATIONS_PATH")
	if err != nil {
		return nil, fmt.Errorf("invalid TRANSLATIONS_PATH: %w", err)
	}
	return paths, nil
}

func parseBaseLang() (string, error) {
	baseLang := os.Getenv("BASE_LANG")
	if baseLang == "" {
		return "", fmt.Errorf("BASE_LANG is not set or is empty")
	}
	return baseLang, nil
}

func parseNamePattern() (string, error) {
	namePattern, err := normalizers.NormalizeOptionalNamePattern(os.Getenv("NAME_PATTERN"))
	if err != nil {
		return "", err
	}
	return namePattern, nil
}

func parseFileExtensions() ([]string, error) {
	fileExts, err := normalizers.NormalizeFileExtensions(parsers.ParseStringArrayEnv("FILE_EXT"))
	if err != nil {
		return nil, fmt.Errorf("invalid FILE_EXT: %w", err)
	}
	return fileExts, nil
}

func parseFlatNaming() (bool, error) {
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		return false, fmt.Errorf("invalid FLAT_NAMING: expected true or false")
	}
	return flatNaming, nil
}
