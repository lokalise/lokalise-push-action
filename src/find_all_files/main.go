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
//
// Optionally, a custom NAME_PATTERN can override both layouts.
// Results are exported via GitHub Actions outputs:
//   - ALL_FILES: comma-separated file list
//   - has_files: true/false
//
// Note: ALL_FILES is comma-separated for downstream shell processing.
// This means file names containing commas are not safely representable
// in that output format.

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

func main() {
	// Read and validate required env variables.
	translationsPaths, baseLang, fileExts, namePattern := validateEnvironment()

	// Parse FLAT_NAMING:
	//   true  -> flat files at the root of each translations path
	//   false -> nested files under <root>/<baseLang>/
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		returnWithError("invalid value for FLAT_NAMING environment variable; expected true or false")
	}

	// Discover files according to the selected strategy.
	allFiles, err := findAllTranslationFiles(translationsPaths, flatNaming, baseLang, fileExts, namePattern)
	if err != nil {
		returnWithError(fmt.Sprintf("unable to find translation files: %v", err))
	}

	// Write outputs for downstream workflow steps.
	processAllFiles(allFiles, githuboutput.WriteToGitHubOutput)
}

// validateEnvironment enforces presence of required inputs and normalizes them.
func validateEnvironment() ([]string, string, []string, string) {
	paths := parseTranslationsPaths()
	baseLang := parseBaseLang()
	namePattern := normalizeNamePattern(os.Getenv("NAME_PATTERN"))
	fileExts := normalizeFileExtensions(parsers.ParseStringArrayEnv("FILE_EXT"))

	return paths, baseLang, fileExts, namePattern
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

// normalizeNamePattern validates an optional custom name pattern.
// The pattern must remain repo-relative and must not escape the repo root.
func normalizeNamePattern(namePattern string) string {
	if namePattern == "" {
		return ""
	}

	np, err := ensureRepoRelative(namePattern)
	if err != nil {
		returnWithError(fmt.Sprintf("invalid NAME_PATTERN %q: %v", namePattern, err))
	}

	return np
}

// normalizeFileExtensions normalizes FILE_EXT values by:
//   - trimming whitespace
//   - removing a leading dot
//   - lowercasing
//   - deduplicating
func normalizeFileExtensions(exts []string) []string {
	if len(exts) == 0 {
		returnWithError("Cannot infer file extension. Make sure FILE_EXT environment variable is set")
	}

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

	return norm
}

// processAllFiles emits GitHub Action outputs.
func processAllFiles(allFiles []string, writeOutput func(key, value string) bool) {
	if len(allFiles) > 0 {
		allFilesStr := strings.Join(allFiles, ",")
		if !writeOutput("ALL_FILES", allFilesStr) || !writeOutput("has_files", "true") {
			returnWithError("cannot write to GITHUB_OUTPUT")
		}
		return
	}

	if !writeOutput("has_files", "false") {
		returnWithError("cannot write to GITHUB_OUTPUT")
	}
}

// findAllTranslationFiles scans each configured root using the chosen strategy.
// Rules:
//   - NAME_PATTERN (if provided) overrides layout rules and is treated as a glob under the root.
//   - Flat:   collect "<root>/<baseLang>.<ext>" if present.
//   - Nested: walk "<root>/<baseLang>" and collect files ending with ".<ext>".
func findAllTranslationFiles(paths []string, flatNaming bool, baseLang string, fileExts []string, namePattern string) ([]string, error) {
	collector := newFileCollector()

	for _, root := range paths {
		if root == "" {
			continue
		}

		var err error
		switch {
		case namePattern != "":
			err = collectFilesByPattern(root, namePattern, collector.add)
		case flatNaming:
			err = collectFlatFiles(root, baseLang, fileExts, collector.add)
		default:
			err = collectNestedFiles(root, baseLang, fileExts, collector.add)
		}

		if err != nil {
			return nil, err
		}
	}

	files := collector.sorted()
	fmt.Fprintf(os.Stderr, "Found %d unique files\n", len(files))

	return files, nil
}

// collectFilesByPattern applies NAME_PATTERN relative to the given root.
// The pattern is evaluated against os.DirFS("."), so it must be repo-relative
// and must not start with "./".
func collectFilesByPattern(root, namePattern string, add func(string)) error {
	pattern := filepath.ToSlash(filepath.Join(root, namePattern))
	pattern = strings.TrimPrefix(pattern, "./")

	globOpts := []doublestar.GlobOption{
		doublestar.WithFilesOnly(),
		doublestar.WithFailOnIOErrors(),
	}

	matches, err := doublestar.Glob(os.DirFS("."), pattern, globOpts...)
	if err != nil {
		return fmt.Errorf("apply name pattern %q: %w", pattern, err)
	}

	for _, match := range matches {
		add(match)
	}

	return nil
}

// collectFlatFiles checks for exact flat-layout file names:
//
//	<root>/<baseLang>.<ext>
//
// Missing files are ignored. Unexpected stat errors are returned.
func collectFlatFiles(root, baseLang string, fileExts []string, add func(string)) error {
	for _, ext := range fileExts {
		target := filepath.Join(root, fmt.Sprintf("%s.%s", baseLang, ext))

		info, err := os.Stat(target)
		if err == nil {
			if !info.IsDir() {
				add(target)
			}
			continue
		}

		if !os.IsNotExist(err) {
			return fmt.Errorf("error accessing file %s: %v", target, err)
		}
	}

	return nil
}

// collectNestedFiles walks the nested layout directory:
//
//	<root>/<baseLang>/...
//
// Missing language directories are treated as "no files found", not as errors.
func collectNestedFiles(root, baseLang string, fileExts []string, add func(string)) error {
	targetDir := filepath.Join(root, baseLang)

	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error accessing directory %s: %v", targetDir, err)
	}

	if !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(targetDir, func(fp string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("error walking through directory %s: %v", targetDir, walkErr)
		}
		if d.IsDir() {
			return nil
		}
		if hasMatchingExtension(d.Name(), fileExts) {
			add(fp)
		}
		return nil
	})
}

// hasMatchingExtension reports whether the file name ends with one of the allowed extensions.
// Comparison is case-insensitive.
func hasMatchingExtension(name string, fileExts []string) bool {
	for _, ext := range fileExts {
		if strings.EqualFold(filepath.Ext(name), "."+ext) {
			return true
		}
	}
	return false
}

// fileCollector accumulates unique file paths and normalizes them to forward slashes
// to keep output deterministic across operating systems.
type fileCollector struct {
	seen  map[string]struct{}
	files []string
}

func newFileCollector() *fileCollector {
	return &fileCollector{
		seen: make(map[string]struct{}),
	}
}

func (c *fileCollector) add(path string) {
	path = filepath.ToSlash(path)
	if _, ok := c.seen[path]; ok {
		return
	}
	c.seen[path] = struct{}{}
	c.files = append(c.files, path)
}

func (c *fileCollector) sorted() []string {
	out := append([]string(nil), c.files...)
	sort.Strings(out)
	return out
}

// ensureRepoRelative validates that the provided path is repo-relative
// and does not escape the repository root.
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

// returnWithError prints an error and exits with a non-zero code.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
