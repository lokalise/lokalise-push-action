package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
)

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

func main() {
	// Read and validate inputs from the environment.
	// This step makes sure we have enough info to derive a set of pathspecs.
	translationsPaths, baseLang, fileExts, namePattern := validateEnvironment()

	// FLAT_NAMING determines whether translations are flat (e.g., locales/en.json)
	// or nested by language (e.g., locales/en/**/*.json).
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		returnWithError("invalid value for FLAT_NAMING environment variable; expected true or false")
	}

	// We persist the generated pathspecs to a file that is later consumed by
	// tj-actions/changed-files via `files_from_source_file`.
	file := createOutputFile()
	defer closeOutputFile(file)

	// Emit one pathspec per line. Consumers expect newline-separated patterns.
	// Each line can be a direct file path or a glob (git pathspec-style).
	if err := storeTranslationPaths(translationsPaths, flatNaming, baseLang, fileExts, namePattern, file); err != nil {
		returnWithError(fmt.Sprintf("cannot store translation paths: %v", err))
	}
}

// validateEnvironment reads required variables and applies simple inference.
// Returns: (paths, base language code, file extensions, optional custom name pattern).
func validateEnvironment() ([]string, string, []string, string) {
	paths, err := parsers.ParseRepoRelativePathsEnv("TRANSLATIONS_PATH")
	if err != nil {
		returnWithError(fmt.Sprintf("failed to process params: %v", err))
	}

	baseLang := os.Getenv("BASE_LANG")
	if baseLang == "" {
		returnWithError("BASE_LANG is not set or is empty")
	}

	namePattern := normalizeNamePattern(os.Getenv("NAME_PATTERN"))
	fileExts := normalizeFileExtensions(parsers.ParseStringArrayEnv("FILE_EXT"))

	return paths, baseLang, fileExts, namePattern
}

// normalizeNamePattern validates an optional custom pattern and keeps it repo-relative.
// Empty input is allowed and returned as-is.
func normalizeNamePattern(pattern string) string {
	if pattern == "" {
		return ""
	}

	// Forbid absolute paths and path traversal outside the repo root.
	normalized, err := ensureRepoRelative(pattern)
	if err != nil {
		returnWithError(fmt.Sprintf("invalid NAME_PATTERN %q: %v", pattern, err))
	}

	return normalized
}

// normalizeFileExtensions supports single or multiple FILE_EXT values,
// normalizes them, and removes duplicates while preserving behavior.
func normalizeFileExtensions(exts []string) []string {
	if len(exts) == 0 {
		returnWithError("Cannot infer file extension. Make sure FILE_EXT environment variable is set")
	}

	seen := make(map[string]struct{}, len(exts))
	normalized := make([]string, 0, len(exts))

	for _, ext := range exts {
		ext = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(ext, ".")))
		if ext == "" {
			continue
		}
		if _, ok := seen[ext]; ok {
			continue
		}
		seen[ext] = struct{}{}
		normalized = append(normalized, ext)
	}

	if len(normalized) == 0 {
		returnWithError("no valid file extensions after normalization")
	}

	return normalized
}

// createOutputFile creates the temp file consumed later by changed-files.
func createOutputFile() *os.File {
	file, err := os.Create("lok_action_paths_temp.txt")
	if err != nil {
		returnWithError(fmt.Sprintf("cannot create output file: %v", err))
	}

	return file
}

// closeOutputFile closes the output file and prints a warning on failure.
// This warning is non-fatal because the file may have already been written successfully.
func closeOutputFile(file *os.File) {
	if cerr := file.Close(); cerr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close file properly: %v\n", cerr)
	}
}

// storeTranslationPaths emits one pathspec per root and (if applicable) per extension.
// Output is newline-separated, ready for consumption by changed-files (files_from_source_file).
// Rules:
//   - If namePattern is set, it fully overrides defaults and is written once per root.
//     The pattern may include globs (e.g., "**/*.yaml") and/or a concrete filename.
//   - If flatNaming is true  -> "<root>/<baseLang>.<ext>"
//   - If flatNaming is false -> "<root>/<baseLang>/**/*.ext"
func storeTranslationPaths(paths []string, flatNaming bool, baseLang string, fileExts []string, namePattern string, writer io.Writer) error {
	seen := make(map[string]struct{}) // avoid duplicates across roots/exts

	// Sort once to keep output deterministic across runs.
	exts := append([]string(nil), fileExts...)
	sort.Strings(exts)

	for _, root := range paths {
		if namePattern != "" {
			// Custom pattern takes precedence; caller is responsible for including
			// filename/ext or globs. We don't expand it per-extension.
			if err := writeUniqueLine(writer, seen, filepath.Join(root, namePattern)); err != nil {
				return err
			}
			continue
		}

		// Generate per-extension patterns based on layout.
		for _, ext := range exts {
			ext = strings.TrimSpace(ext)
			if ext == "" {
				continue
			}

			pattern := buildTranslationPattern(root, flatNaming, baseLang, ext)
			if err := writeUniqueLine(writer, seen, pattern); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildTranslationPattern builds the pathspec for a single root/extension pair.
func buildTranslationPattern(root string, flatNaming bool, baseLang, ext string) string {
	if flatNaming {
		// <root>/<baseLang>.<ext>
		return filepath.Join(root, fmt.Sprintf("%s.%s", baseLang, ext))
	}

	// <root>/<baseLang>/**/*.ext
	return filepath.Join(root, baseLang, "**", fmt.Sprintf("*.%s", ext))
}

// writeUniqueLine writes a normalized newline-terminated pathspec once.
func writeUniqueLine(writer io.Writer, seen map[string]struct{}, path string) error {
	// Normalize to forward slashes for cross-platform consistency and
	// anchor to repo root with a leading "./" (helps avoid CWD surprises).
	line := filepath.ToSlash(filepath.Join(".", path))

	if _, ok := seen[line]; ok {
		return nil
	}
	seen[line] = struct{}{}

	_, err := writer.Write([]byte(line + "\n"))
	return err
}

// ensureRepoRelative validates that the path is relative to the repo root
// and does not escape it via absolute paths or parent traversal.
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

// returnWithError prints an error and exits non-zero.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
