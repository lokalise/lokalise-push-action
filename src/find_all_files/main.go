package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/bodrovis/lokalise-actions-common/v2/githuboutput"
	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
)

// This program discovers translation files based on env configuration.
// It supports two layout styles:
//   - flat:   <root>/<baseLang>.<ext>
//   - nested: <root>/<baseLang>/**/<anything>.<ext>
// Optionally, a custom NAME_PATTERN can override both.
// Results are exported via GitHub Actions outputs: ALL_FILES (comma-separated) and has_files (true/false).

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

func main() {
	// Read and validate required env variables.
	translationsPaths, baseLang, fileExts, namePattern := validateEnvironment()

	// Parse FLAT_NAMING: true -> flat files at root; false -> nested per-language directories.
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		returnWithError("invalid value for FLAT_NAMING environment variable; expected true or false")
	}

	// Discover files according to the selected strategy.
	allFiles, err := findAllTranslationFiles(translationsPaths, flatNaming, baseLang, fileExts, namePattern)
	if err != nil {
		returnWithError(fmt.Sprintf("unable to find translation files: %v", err))
	}

	// Write outputs for downstream steps.
	processAllFiles(allFiles, githuboutput.WriteToGitHubOutput)
}

// validateEnvironment enforces presence of required inputs and performs simple inference (FILE_EXT ← FILE_FORMAT).
func validateEnvironment() ([]string, string, []string, string) {
	translationsPaths := parsers.ParseStringArrayEnv("TRANSLATIONS_PATH")
	if len(translationsPaths) == 0 {
		returnWithError("TRANSLATIONS_PATH is not set or is empty")
	}

	cleanedRoots := make([]string, 0, len(translationsPaths))
	for _, r := range translationsPaths {
		rr, err := ensureRepoRelative(r)
		if err != nil {
			returnWithError(fmt.Sprintf("invalid TRANSLATIONS_PATH %q: %v", r, err))
		}
		cleanedRoots = append(cleanedRoots, rr)
	}

	baseLang := os.Getenv("BASE_LANG")
	if baseLang == "" {
		returnWithError("BASE_LANG is not set or is empty")
	}

	namePattern := os.Getenv("NAME_PATTERN")
	if namePattern != "" {
		np, err := ensureRepoRelative(namePattern)
		if err != nil {
			returnWithError(fmt.Sprintf("invalid NAME_PATTERN %q: %v", namePattern, err))
		}
		namePattern = np
	}

	exts := parsers.ParseStringArrayEnv("FILE_EXT")
	if len(exts) == 0 {
		if v := os.Getenv("FILE_FORMAT"); v != "" {
			exts = []string{v}
		}
	}
	if len(exts) == 0 {
		returnWithError("Cannot infer file extension. Make sure FILE_FORMAT or FILE_EXT environment variables are set")
	}

	// normalize + dedupe
	seen := make(map[string]struct{}, len(exts))
	norm := make([]string, 0, len(exts))
	for _, e := range exts {
		e = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(e, ".")))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		norm = append(norm, e)
	}
	if len(norm) == 0 {
		returnWithError("no valid file extensions after normalization")
	}

	return cleanedRoots, baseLang, norm, namePattern
}

// processAllFiles emits GitHub Action outputs.
// Note: ALL_FILES is a comma-separated list (consumers must handle paths with spaces properly).
func processAllFiles(allFiles []string, writeOutput func(key, value string) bool) {
	if len(allFiles) > 0 {
		allFilesStr := strings.Join(allFiles, ",")
		if !writeOutput("ALL_FILES", allFilesStr) || !writeOutput("has_files", "true") {
			returnWithError("cannot write to GITHUB_OUTPUT")
		}
	} else {
		if !writeOutput("has_files", "false") {
			returnWithError("cannot write to GITHUB_OUTPUT")
		}
	}
}

// findAllTranslationFiles scans each configured root using the chosen strategy.
// - NAME_PATTERN (if provided) overrides layout rules and is treated as a glob under the root.
// - Flat: single file "<root>/<baseLang>.<ext>" if present.
// - Nested: walk "<root>/<baseLang>" and collect files ending with ".<ext>".
func findAllTranslationFiles(paths []string, flatNaming bool, baseLang string, fileExts []string, namePattern string) ([]string, error) {
	var allFiles []string
	seen := make(map[string]struct{})

	add := func(p string) {
		p = filepath.ToSlash(p)
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		allFiles = append(allFiles, p)
	}

	for _, path := range paths {
		if path == "" {
			continue
		}

		if namePattern != "" {
			pattern := filepath.ToSlash(filepath.Join(path, namePattern))
			pattern = strings.TrimPrefix(pattern, "./") // doublestar on DirFS(".") wants relative pattern

			matches, err := doublestar.Glob(os.DirFS("."), pattern)
			if err != nil {
				return nil, fmt.Errorf("error applying name pattern %s: %v", pattern, err)
			}

			for _, m := range matches {
				add(m)
			}

			continue
		}

		if flatNaming {
			for _, ext := range fileExts {
				target := filepath.Join(path, fmt.Sprintf("%s.%s", baseLang, ext))
				if info, err := os.Stat(target); err == nil && !info.IsDir() {
					add(target)
				} else if err != nil && !os.IsNotExist(err) {
					return nil, fmt.Errorf("error accessing file %s: %v", target, err)
				}
			}
			continue
		}

		// nested
		targetDir := filepath.Join(path, baseLang)
		if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
			err := filepath.WalkDir(targetDir, func(fp string, d os.DirEntry, err error) error {
				if err != nil {
					return fmt.Errorf("error walking through directory %s: %v", targetDir, err)
				}
				if d.IsDir() {
					return nil
				}
				name := d.Name()
				for _, ext := range fileExts {
					if strings.EqualFold(filepath.Ext(name), "."+ext) {
						add(fp)
						break
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("error accessing directory %s: %v", targetDir, err)
		}
	}

	fmt.Fprintf(os.Stderr, "Found %d unique files\n", len(allFiles))
	sort.Strings(allFiles)

	return allFiles, nil
}

func ensureRepoRelative(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("empty path")
	}

	clean := filepath.Clean(p)

	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("path must be relative to repo: %q", p)
	}

	s := filepath.ToSlash(clean)
	if strings.HasPrefix(s, "/") {
		return "", fmt.Errorf("path must be relative to repo: %q", p)
	}

	if s == ".." || strings.HasPrefix(s, "../") {
		return "", fmt.Errorf("path escapes repo root: %q", p)
	}

	return clean, nil
}

func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
